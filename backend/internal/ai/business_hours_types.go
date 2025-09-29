package ai

// BusinessHours representa os horários de funcionamento da loja
type BusinessHours struct {
	Monday    DayHours `json:"monday"`
	Tuesday   DayHours `json:"tuesday"`
	Wednesday DayHours `json:"wednesday"`
	Thursday  DayHours `json:"thursday"`
	Friday    DayHours `json:"friday"`
	Saturday  DayHours `json:"saturday"`
	Sunday    DayHours `json:"sunday"`
	Timezone  string   `json:"timezone"`
}

type DayHours struct {
	Enabled bool   `json:"enabled"`
	Open    string `json:"open"`
	Close   string `json:"close"`
}
