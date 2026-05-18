package domain

type Counter string

const (
	CounterAPICalls        Counter = "api_calls"
	CounterEventsIngested  Counter = "events_ingested"
	CounterIncidentsOpened Counter = "incidents_opened"
	CounterActionsExecuted Counter = "actions_executed"
)

type UsageSummary struct {
	Period   string           `json:"period"`
	Counters map[string]int64 `json:"counters"`
}
