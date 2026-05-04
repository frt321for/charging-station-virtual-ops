package site

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"charging-ops/backend/internal/platform/database"
)

// ErrNotFound is returned when a site cannot be found.
var ErrNotFound = errors.New("site not found")

// Repository reads site and charger topology data from PostgreSQL.
type Repository struct {
	database *database.Client
}

// NewRepository creates a site repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// ListSummaries returns site overview rows with operational counters.
func (r *Repository) ListSummaries(ctx context.Context) ([]Summary, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT
			s.id::text,
			s.code,
			s.name,
			s.campus,
			s.capacity_kw,
			s.load_limit_kw,
			s.status,
			COUNT(DISTINCT c.id) FILTER (WHERE c.deleted_at IS NULL)::int AS charger_count,
			COUNT(DISTINCT cn.id) FILTER (WHERE cn.deleted_at IS NULL)::int AS connector_count,
			COUNT(DISTINCT cn.id) FILTER (WHERE cn.status = 'available' AND cn.deleted_at IS NULL)::int AS available_connectors,
			COUNT(DISTINCT cs.id) FILTER (
				WHERE cs.status IN ('starting', 'charging', 'paused') AND cs.deleted_at IS NULL
			)::int AS active_sessions,
			COUNT(DISTINCT c.id) FILTER (WHERE c.status = 'fault' AND c.deleted_at IS NULL)::int AS faulted_chargers
		FROM sites s
		LEFT JOIN charger_groups g ON g.site_id = s.id AND g.deleted_at IS NULL
		LEFT JOIN chargers c ON c.group_id = g.id AND c.deleted_at IS NULL
		LEFT JOIN connectors cn ON cn.charger_id = c.id AND cn.deleted_at IS NULL
		LEFT JOIN charging_sessions cs ON cs.site_id = s.id AND cs.deleted_at IS NULL
		WHERE s.deleted_at IS NULL
		GROUP BY s.id
		ORDER BY s.code
	`)
	if err != nil {
		return nil, fmt.Errorf("query site summaries: %w", err)
	}
	defer rows.Close()

	summaries := make([]Summary, 0)
	for rows.Next() {
		var summary Summary
		if err := rows.Scan(
			&summary.ID,
			&summary.Code,
			&summary.Name,
			&summary.Campus,
			&summary.CapacityKW,
			&summary.LoadLimitKW,
			&summary.Status,
			&summary.ChargerCount,
			&summary.ConnectorCount,
			&summary.AvailableConnectors,
			&summary.ActiveSessions,
			&summary.FaultedChargers,
		); err != nil {
			return nil, fmt.Errorf("scan site summary: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate site summaries: %w", err)
	}

	return summaries, nil
}

// GetTopology returns the nested topology for a site ID or site code.
func (r *Repository) GetTopology(ctx context.Context, siteID string) (Topology, error) {
	rows, err := r.database.Pool().Query(ctx, topologySQL, siteID)
	if err != nil {
		return Topology{}, fmt.Errorf("query site topology: %w", err)
	}
	defer rows.Close()

	builder := newTopologyBuilder()
	for rows.Next() {
		row, err := scanTopologyRow(rows)
		if err != nil {
			return Topology{}, err
		}
		builder.add(row)
	}
	if err := rows.Err(); err != nil {
		return Topology{}, fmt.Errorf("iterate site topology: %w", err)
	}
	if !builder.found {
		return Topology{}, ErrNotFound
	}

	return builder.topology, nil
}

const topologySQL = `
	SELECT
		s.id::text,
		s.code,
		s.name,
		s.campus,
		s.capacity_kw,
		s.load_limit_kw,
		s.status,
		a.id::text,
		a.code,
		a.name,
		a.load_limit_kw,
		a.status,
		g.id::text,
		g.code,
		g.name,
		g.electrical_node,
		g.load_limit_kw,
		g.priority,
		g.status,
		c.id::text,
		c.code,
		c.name,
		c.charger_type,
		c.rated_power_kw,
		c.status,
		c.installation_location,
		cn.id::text,
		cn.code,
		cn.connector_no,
		cn.max_power_kw,
		cn.status
	FROM sites s
	LEFT JOIN areas a ON a.site_id = s.id AND a.deleted_at IS NULL
	LEFT JOIN charger_groups g ON g.area_id = a.id AND g.deleted_at IS NULL
	LEFT JOIN chargers c ON c.group_id = g.id AND c.deleted_at IS NULL
	LEFT JOIN connectors cn ON cn.charger_id = c.id AND cn.deleted_at IS NULL
	WHERE s.deleted_at IS NULL AND (s.id::text = $1 OR s.code = $1)
	ORDER BY a.sort_order, a.code, g.priority, g.code, c.code, cn.connector_no
`

type topologyRow struct {
	site      SiteNode
	area      nullableArea
	group     nullableGroup
	charger   nullableCharger
	connector nullableConnector
}

type nullableArea struct {
	node AreaNode
	id   sql.NullString
}

type nullableGroup struct {
	node GroupNode
	id   sql.NullString
}

type nullableCharger struct {
	node ChargerNode
	id   sql.NullString
}

type nullableConnector struct {
	node ConnectorNode
	id   sql.NullString
}

type topologyScanner interface {
	Scan(dest ...any) error
}

func scanTopologyRow(scanner topologyScanner) (topologyRow, error) {
	var row topologyRow
	var areaCode, areaName, areaStatus sql.NullString
	var areaLoad sql.NullFloat64
	var groupCode, groupName, groupNode, groupStatus sql.NullString
	var groupLoad sql.NullFloat64
	var groupPriority sql.NullInt64
	var chargerCode, chargerName, chargerType, chargerStatus, chargerLocation sql.NullString
	var chargerPower sql.NullFloat64
	var connectorCode, connectorStatus sql.NullString
	var connectorNo sql.NullInt64
	var connectorPower sql.NullFloat64

	err := scanner.Scan(
		&row.site.ID,
		&row.site.Code,
		&row.site.Name,
		&row.site.Campus,
		&row.site.CapacityKW,
		&row.site.LoadLimitKW,
		&row.site.Status,
		&row.area.id,
		&areaCode,
		&areaName,
		&areaLoad,
		&areaStatus,
		&row.group.id,
		&groupCode,
		&groupName,
		&groupNode,
		&groupLoad,
		&groupPriority,
		&groupStatus,
		&row.charger.id,
		&chargerCode,
		&chargerName,
		&chargerType,
		&chargerPower,
		&chargerStatus,
		&chargerLocation,
		&row.connector.id,
		&connectorCode,
		&connectorNo,
		&connectorPower,
		&connectorStatus,
	)
	if err != nil {
		return topologyRow{}, fmt.Errorf("scan site topology: %w", err)
	}

	row.area.node = AreaNode{
		ID:          row.area.id.String,
		Code:        areaCode.String,
		Name:        areaName.String,
		LoadLimitKW: areaLoad.Float64,
		Status:      areaStatus.String,
	}
	row.group.node = GroupNode{
		ID:             row.group.id.String,
		Code:           groupCode.String,
		Name:           groupName.String,
		ElectricalNode: groupNode.String,
		LoadLimitKW:    groupLoad.Float64,
		Priority:       int(groupPriority.Int64),
		Status:         groupStatus.String,
	}
	row.charger.node = ChargerNode{
		ID:                   row.charger.id.String,
		Code:                 chargerCode.String,
		Name:                 chargerName.String,
		ChargerType:          chargerType.String,
		RatedPowerKW:         chargerPower.Float64,
		Status:               chargerStatus.String,
		InstallationLocation: chargerLocation.String,
	}
	row.connector.node = ConnectorNode{
		ID:         row.connector.id.String,
		Code:       connectorCode.String,
		Number:     int(connectorNo.Int64),
		MaxPowerKW: connectorPower.Float64,
		Status:     connectorStatus.String,
	}

	return row, nil
}

type topologyBuilder struct {
	topology     Topology
	found        bool
	areaIndexes  map[string]int
	groupIndexes map[string]groupIndex
	chargerIndex map[string]chargerIndex
}

type groupIndex struct {
	area  int
	group int
}

type chargerIndex struct {
	area    int
	group   int
	charger int
}

func newTopologyBuilder() *topologyBuilder {
	return &topologyBuilder{
		areaIndexes:  make(map[string]int),
		groupIndexes: make(map[string]groupIndex),
		chargerIndex: make(map[string]chargerIndex),
	}
}

func (b *topologyBuilder) add(row topologyRow) {
	if !b.found {
		b.topology.Site = row.site
		b.found = true
	}
	if !row.area.id.Valid {
		return
	}

	areaPosition := b.ensureArea(row.area.node)
	if !row.group.id.Valid {
		return
	}

	groupPosition := b.ensureGroup(areaPosition, row.group.node)
	if !row.charger.id.Valid {
		return
	}

	chargerPosition := b.ensureCharger(areaPosition, groupPosition, row.charger.node)
	if !row.connector.id.Valid {
		return
	}

	charger := &b.topology.Areas[areaPosition].Groups[groupPosition].Chargers[chargerPosition]
	charger.Connectors = append(charger.Connectors, row.connector.node)
}

func (b *topologyBuilder) ensureArea(area AreaNode) int {
	if index, ok := b.areaIndexes[area.ID]; ok {
		return index
	}
	b.topology.Areas = append(b.topology.Areas, area)
	index := len(b.topology.Areas) - 1
	b.areaIndexes[area.ID] = index
	return index
}

func (b *topologyBuilder) ensureGroup(areaPosition int, group GroupNode) int {
	if index, ok := b.groupIndexes[group.ID]; ok {
		return index.group
	}
	area := &b.topology.Areas[areaPosition]
	area.Groups = append(area.Groups, group)
	index := len(area.Groups) - 1
	b.groupIndexes[group.ID] = groupIndex{area: areaPosition, group: index}
	return index
}

func (b *topologyBuilder) ensureCharger(areaPosition int, groupPosition int, charger ChargerNode) int {
	if index, ok := b.chargerIndex[charger.ID]; ok {
		return index.charger
	}
	group := &b.topology.Areas[areaPosition].Groups[groupPosition]
	group.Chargers = append(group.Chargers, charger)
	index := len(group.Chargers) - 1
	b.chargerIndex[charger.ID] = chargerIndex{area: areaPosition, group: groupPosition, charger: index}
	return index
}
