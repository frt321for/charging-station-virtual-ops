package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charging-ops/backend/internal/common/response"
	"charging-ops/backend/internal/domain/auth"
)

type repository interface {
	FindSessionContext(ctx context.Context, sessionID string) (sessionContext, error)
	FindWorkOrderContext(ctx context.Context, workOrderID string) (workOrderContext, error)
	FindSiteContext(ctx context.Context, siteID string) (siteContext, error)
}

// Service coordinates read-only AI assistant workflows.
type Service struct {
	repository repository
	generator  Generator
}

// NewService creates an AI operations assistant service.
func NewService(repository repository, generator Generator) *Service {
	return &Service{repository: repository, generator: generator}
}

// ExplainSession generates an abnormal session explanation.
func (s *Service) ExplainSession(ctx context.Context, params SessionExplanationParams) (Insight, error) {
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return Insight{}, fmt.Errorf("%w: missing session id", ErrInvalidRequest)
	}
	data, err := s.repository.FindSessionContext(ctx, sessionID)
	if err != nil {
		return Insight{}, err
	}
	if !auth.CanAccessSite(ctx, data.Site.ID, data.Site.Code) {
		return Insight{}, auth.ErrForbidden
	}
	return s.generate(ctx, generateParams{
		kind:        KindSessionExplanation,
		site:        data.Site,
		target:      TargetRef{Type: "session", ID: data.Session.ID, Code: data.Session.SessionNo},
		contextData: data,
		evidence:    sessionEvidence(data),
		suggestions: sessionSuggestions(data),
		fallback:    fallbackSessionExplanation(data),
	})
}

// SummarizeWorkOrder generates a maintenance handoff summary.
func (s *Service) SummarizeWorkOrder(ctx context.Context, params WorkOrderSummaryParams) (Insight, error) {
	workOrderID := strings.TrimSpace(params.WorkOrderID)
	if workOrderID == "" {
		return Insight{}, fmt.Errorf("%w: missing work order id", ErrInvalidRequest)
	}
	data, err := s.repository.FindWorkOrderContext(ctx, workOrderID)
	if err != nil {
		return Insight{}, err
	}
	if !auth.CanAccessSite(ctx, data.Site.ID, data.Site.Code) {
		return Insight{}, auth.ErrForbidden
	}
	return s.generate(ctx, generateParams{
		kind:        KindWorkOrderSummary,
		site:        data.Site,
		target:      TargetRef{Type: "work_order", ID: data.WorkOrder.ID, Code: data.WorkOrder.WorkOrderNo},
		contextData: data,
		evidence:    workOrderEvidence(data),
		suggestions: workOrderSuggestions(data),
		fallback:    fallbackWorkOrderSummary(data),
	})
}

// PredictCongestion generates a site congestion risk note.
func (s *Service) PredictCongestion(ctx context.Context, params CongestionRiskParams) (Insight, error) {
	data, err := s.siteContext(ctx, params.SiteID)
	if err != nil {
		return Insight{}, err
	}
	horizon := params.HorizonHours
	if horizon <= 0 {
		horizon = 4
	}
	return s.generate(ctx, generateParams{
		kind:        KindCongestionRisk,
		site:        data.Site,
		target:      TargetRef{Type: "site", ID: data.Site.ID, Code: data.Site.Code},
		contextData: map[string]any{"site": data, "horizonHours": horizon},
		evidence:    siteEvidence(data),
		suggestions: congestionSuggestions(data),
		fallback:    fallbackCongestionRisk(data, horizon),
	})
}

// DraftDailyReport generates a station daily report draft.
func (s *Service) DraftDailyReport(ctx context.Context, params DailyReportParams) (Insight, error) {
	data, err := s.siteContext(ctx, params.SiteID)
	if err != nil {
		return Insight{}, err
	}
	businessDate := strings.TrimSpace(params.BusinessDate)
	if businessDate == "" {
		businessDate = time.Now().Format("2006-01-02")
	}
	return s.generate(ctx, generateParams{
		kind:        KindDailyReport,
		site:        data.Site,
		target:      TargetRef{Type: "site", ID: data.Site.ID, Code: data.Site.Code},
		contextData: map[string]any{"site": data, "businessDate": businessDate},
		evidence:    siteEvidence(data),
		suggestions: reportSuggestions(data),
		fallback:    fallbackDailyReport(data, businessDate),
	})
}

// AnswerStationQuestion answers a bounded station question.
func (s *Service) AnswerStationQuestion(ctx context.Context, params StationQAParams) (Insight, error) {
	question := strings.TrimSpace(params.Question)
	if question == "" {
		return Insight{}, fmt.Errorf("%w: missing question", ErrInvalidRequest)
	}
	data, err := s.siteContext(ctx, params.SiteID)
	if err != nil {
		return Insight{}, err
	}
	return s.generate(ctx, generateParams{
		kind:        KindStationQA,
		site:        data.Site,
		target:      TargetRef{Type: "site", ID: data.Site.ID, Code: data.Site.Code},
		contextData: map[string]any{"site": data, "question": question},
		evidence:    siteEvidence(data),
		suggestions: stationQASuggestions(data),
		fallback:    fallbackStationAnswer(data, question),
	})
}

func (s *Service) siteContext(ctx context.Context, siteID string) (siteContext, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID != "" && !auth.CanAccessSite(ctx, siteID, siteID) {
		return siteContext{}, auth.ErrForbidden
	}
	data, err := s.repository.FindSiteContext(ctx, siteID)
	if err != nil {
		return siteContext{}, err
	}
	if !auth.CanAccessSite(ctx, data.Site.ID, data.Site.Code) {
		return siteContext{}, auth.ErrForbidden
	}
	return data, nil
}

type generateParams struct {
	kind        InsightKind
	site        SiteRef
	target      TargetRef
	contextData any
	evidence    []Evidence
	suggestions []string
	fallback    string
}

func (s *Service) generate(ctx context.Context, params generateParams) (Insight, error) {
	generated := GeneratedContent{
		Content:  params.fallback,
		Provider: "local-fallback",
		Model:    "deterministic-ops",
	}
	if s.generator != nil {
		if result, err := s.generator.Generate(ctx, buildPrompt(params.kind, params.contextData)); err == nil {
			generated = result
		}
	}
	return Insight{
		RequestID:   response.NewTraceID(),
		Kind:        params.kind,
		Site:        params.site,
		Target:      params.target,
		Content:     generated.Content,
		Suggestions: params.suggestions,
		Evidence:    params.evidence,
		Provider:    generated.Provider,
		Model:       generated.Model,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func buildPrompt(kind InsightKind, data any) Prompt {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		payload = []byte(`{}`)
	}
	return Prompt{
		Kind:   kind,
		System: "你是园区充电桩运营中台的只读 AI 助手。只基于输入 JSON 回答，不编造外部事实，不下发命令，不要求用户去系统外操作。输出中文，直接给业务结论、证据链和建议动作。",
		User:   fmt.Sprintf("任务类型：%s\n上下文 JSON：\n%s", kind, string(payload)),
	}
}

func sessionEvidence(data sessionContext) []Evidence {
	return []Evidence{
		{Label: "会话", Value: data.Session.SessionNo},
		{Label: "状态", Value: data.Session.Status},
		{Label: "事件", Value: fmt.Sprintf("%d 条", len(data.Events))},
		{Label: "读数", Value: fmt.Sprintf("%d 条", len(data.Meters))},
		{Label: "命令", Value: fmt.Sprintf("%d 条", len(data.Commands))},
		{Label: "核查", Value: fmt.Sprintf("%d 条", len(data.Exceptions))},
	}
}

func workOrderEvidence(data workOrderContext) []Evidence {
	return []Evidence{
		{Label: "工单", Value: data.WorkOrder.WorkOrderNo},
		{Label: "等级", Value: data.WorkOrder.Severity},
		{Label: "状态", Value: data.WorkOrder.Status},
		{Label: "桩机", Value: data.WorkOrder.ChargerCode},
		{Label: "事件", Value: fmt.Sprintf("%d 条", len(data.Events))},
	}
}

func siteEvidence(data siteContext) []Evidence {
	return []Evidence{
		{Label: "站点", Value: data.Site.Code},
		{Label: "当前负载", Value: fmt.Sprintf("%.1f / %.1f kW", data.CurrentLoadKW, data.LoadLimitKW)},
		{Label: "充电中", Value: fmt.Sprintf("%d", data.ActiveSessions)},
		{Label: "排队", Value: fmt.Sprintf("%d", data.WaitingSessions)},
		{Label: "故障桩", Value: fmt.Sprintf("%d", data.FaultedChargers)},
		{Label: "核查", Value: fmt.Sprintf("%d", data.OpenExceptions)},
	}
}

func sessionSuggestions(data sessionContext) []string {
	suggestions := []string{"核对事件时间线、远程命令回执和最后两条电表读数。"}
	if len(data.Exceptions) > 0 {
		suggestions = append(suggestions, "保持账单在核查队列内，确认异常原因后再确认账单。")
	}
	if data.Session.StopReason == "" {
		suggestions = append(suggestions, "补齐停止原因，避免对账阶段缺少闭环证据。")
	}
	return suggestions
}

func workOrderSuggestions(data workOrderContext) []string {
	suggestions := []string{"按 SLA 节点推进接单、到场、处理、复测和恢复记录。"}
	if data.WorkOrder.AssigneeName == "" {
		suggestions = append(suggestions, "先补充分派班组，再推进工单状态。")
	}
	if data.WorkOrder.Severity == "high" || data.WorkOrder.Severity == "critical" {
		suggestions = append(suggestions, "保留复测结果和恢复证据，降低重复故障风险。")
	}
	return suggestions
}

func congestionSuggestions(data siteContext) []string {
	ratio := loadRatio(data)
	if ratio >= 0.9 {
		return []string{"暂停低优先级新启动，优先执行限功率和排队策略。", "检查最近负载控制记录是否已经生效。"}
	}
	if ratio >= 0.75 {
		return []string{"关注排队会话和高功率 DC 枪口，提前准备限功率。"}
	}
	return []string{"维持当前策略，重点观察故障枪口和排队变化。"}
}

func reportSuggestions(data siteContext) []string {
	return []string{"日报草稿发布前核对财务核查队列和未关闭工单。", "将负载控制次数、SLA 风险和收入电量作为交班重点。"}
}

func stationQASuggestions(data siteContext) []string {
	if data.OpenWorkOrders > 0 {
		return []string{"追踪未关闭工单的 SLA 节点。"}
	}
	return []string{"继续按站点负载和核查队列做例行巡检。"}
}

func fallbackSessionExplanation(data sessionContext) string {
	latest := "暂无读数"
	if len(data.Meters) > 0 {
		meter := data.Meters[len(data.Meters)-1]
		latest = fmt.Sprintf("最新功率 %.1f kW，电表 %.3f kWh", meter.PowerKW, meter.MeterKWh)
	}
	return fmt.Sprintf(
		"会话 %s 当前为 %s，关联 %s / %s。系统已记录 %d 条事件、%d 条命令和 %d 条电表读数，%s。核查异常 %d 条，停止原因：%s。",
		data.Session.SessionNo,
		data.Session.Status,
		data.Session.ChargerCode,
		data.Session.ConnectorCode,
		len(data.Events),
		len(data.Commands),
		len(data.Meters),
		latest,
		len(data.Exceptions),
		defaultText(data.Session.StopReason, "未填"),
	)
}

func fallbackWorkOrderSummary(data workOrderContext) string {
	return fmt.Sprintf(
		"工单 %s 为 %s 级，状态 %s，目标 %s / %s，标题为“%s”。当前班组：%s，已记录 %d 条工单事件，恢复截止时间 %s。",
		data.WorkOrder.WorkOrderNo,
		data.WorkOrder.Severity,
		data.WorkOrder.Status,
		data.WorkOrder.ChargerCode,
		defaultText(data.WorkOrder.ConnectorCode, "--"),
		data.WorkOrder.Title,
		defaultText(data.WorkOrder.AssigneeName, "未分派"),
		len(data.Events),
		data.WorkOrder.RecoveryDueAt.Format(time.RFC3339),
	)
}

func fallbackCongestionRisk(data siteContext, horizon int) string {
	level := "低"
	ratio := loadRatio(data)
	if ratio >= 0.9 || data.WaitingSessions >= 3 {
		level = "高"
	} else if ratio >= 0.75 || data.WaitingSessions > 0 {
		level = "中"
	}
	return fmt.Sprintf(
		"%s 未来 %d 小时拥堵风险为%s。当前负载 %.1f/%.1f kW，充电中 %d，会话排队 %d，可用枪口 %d，故障桩 %d。",
		data.Site.Code,
		horizon,
		level,
		data.CurrentLoadKW,
		data.LoadLimitKW,
		data.ActiveSessions,
		data.WaitingSessions,
		data.AvailableConnectors,
		data.FaultedChargers,
	)
}

func fallbackDailyReport(data siteContext, businessDate string) string {
	return fmt.Sprintf(
		"%s %s 日报草稿：当前负载 %.1f kW，站点阈值 %.1f kW；充电中 %d，排队 %d，可用枪口 %d，故障桩 %d；今日电量 %.2f kWh，收入 %.2f 元；未关闭工单 %d，核查异常 %d。",
		data.Site.Code,
		businessDate,
		data.CurrentLoadKW,
		data.LoadLimitKW,
		data.ActiveSessions,
		data.WaitingSessions,
		data.AvailableConnectors,
		data.FaultedChargers,
		data.EnergyTodayKWh,
		data.RevenueToday,
		data.OpenWorkOrders,
		data.OpenExceptions,
	)
}

func fallbackStationAnswer(data siteContext, question string) string {
	return fmt.Sprintf(
		"问题：%s。站点 %s 当前负载 %.1f/%.1f kW，充电中 %d，排队 %d，可用枪口 %d，故障桩 %d，未关闭工单 %d，核查异常 %d。",
		question,
		data.Site.Code,
		data.CurrentLoadKW,
		data.LoadLimitKW,
		data.ActiveSessions,
		data.WaitingSessions,
		data.AvailableConnectors,
		data.FaultedChargers,
		data.OpenWorkOrders,
		data.OpenExceptions,
	)
}

func loadRatio(data siteContext) float64 {
	if data.LoadLimitKW <= 0 {
		return 0
	}
	return data.CurrentLoadKW / data.LoadLimitKW
}

func defaultText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
