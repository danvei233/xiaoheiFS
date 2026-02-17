package shared

type SMSMessage struct {
	TemplateID string
	Content    string
	Vars       map[string]string
	Phones     []string
}

type SMSDelivery struct {
	MessageID string
}
