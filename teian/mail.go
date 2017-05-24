package teian

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
)

type Mail struct {
	To      string
	From    string
	Subject string
	Content string
}

func SendMail(mail Mail) error {
	buf := &bytes.Buffer{}
	if err := mailTmpl.Execute(buf, mail); err != nil {
		return fmt.Errorf("execute template failed: %v", err)
	}

	cmd := exec.Command("bash", "-c", "echo \""+buf.String()+"\" | sendmail kusubooru@gmail.com")
	cmd.Stderr = os.Stderr
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("sendmail failed: %v", err)
	}
	return nil
}

var mailTmpl = template.Must(template.New("mailTemplate").Parse(mailTemplate))

const mailTemplate = `To: {{.To}}
From: {{.From}}
Subject: {{.Subject}}

{{.Content}}
`
