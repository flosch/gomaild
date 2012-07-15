package maild

// Your mail handler will get a Mail instance 
type Mail struct {
	// The hostname from the HELO command
	Hostname   string
	
	// Envelope Sender
	From       string
	
	// Envelope To
	Recipients []string
	
	// Mailcontent, you might want parse this with 
	// the net/mail package
	Data       string
}

func NewMail() *Mail {
	return &Mail{
		Recipients: make([]string, 0, 1),
	}
}
