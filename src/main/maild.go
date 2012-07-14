package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"maild"
)

func mailHandler(mail *maild.Mail) {
	filename := fmt.Sprintf("%d.mail", time.Now().Unix())

	fmt.Printf("Received mail from %s to %+s (saved as %s)\n",
		mail.From, mail.Recipients, filename)

	ioutil.WriteFile(filename, []byte(mail.Data), 0755)
}

var address = flag.String("address", ":2525", "Address running on")

func main() {
	flag.Parse()
	fmt.Printf("GoMaild running on %s\n", *address)
	server := maild.NewMailServer(*address)
	log.Fatal(server.ListenAndReceive(mailHandler))
}
