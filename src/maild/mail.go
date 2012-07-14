package maild

type Mail struct {
	Hostname   string
	From       string
	Recipients []string
	Data       string
}

func NewMail() *Mail {
	return &Mail{
		Recipients: make([]string, 1),
	}
}
