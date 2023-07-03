package bot

type Payload struct {
	UserId string `json:"userId"`
	Prompt string `json:"prompt"`
}

func newPayload(userId string, prompt string) (p Payload) {
	p.UserId = userId
	p.Prompt = prompt
	return
}

type Result struct {
	Text string `json:"text"`
}
