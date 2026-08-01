package notifications

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

// fakeSES captura lo que se le manda a SES. Existe porque el constructor pide una interfaz:
// con el tipo concreto, la única forma de verificar QUÉ se envía es enviarlo de verdad.
type fakeSES struct {
	input *sesv2.SendEmailInput
	calls int
}

func (f *fakeSES) SendEmail(_ context.Context, input *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.input = input
	f.calls++
	return &sesv2.SendEmailOutput{}, nil
}

// TestSESSenderDeclaresUTF8: sin `Charset`, SES interpreta el contenido como ASCII de 7
// bits y cualquier acento llega corrompido. Un correo transaccional en español sale
// ilegible sin que nada falle.
func TestSESSenderDeclaresUTF8(t *testing.T) {
	client := &fakeSES{}
	sender := NewSESSender(client, "no-reply@example.test")

	if err := sender.Send(context.Background(), EmailMessage{
		To:       "paciente@example.test",
		Subject:  "Invitación a la organización",
		TextBody: "Te invitaron a la Clínica del Sur. Confirmá tu dirección.",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	simple := client.input.Content.Simple
	if simple.Subject.Charset == nil || *simple.Subject.Charset != "UTF-8" {
		t.Fatalf("el asunto sale sin charset: %+v", simple.Subject)
	}
	if simple.Body.Text.Charset == nil || *simple.Body.Text.Charset != "UTF-8" {
		t.Fatalf("el cuerpo sale sin charset: %+v", simple.Body.Text)
	}
}

// TestSESSenderOmitsTheEmptyHTMLPart es el defecto más caro de los dos: una parte HTML vacía
// produce un multipart cuya alternativa HTML está en blanco, y los clientes que prefieren
// HTML —casi todos— muestran un correo VACÍO. SES responde 200, así que nada falla: el
// destinatario simplemente recibe nada.
func TestSESSenderOmitsTheEmptyHTMLPart(t *testing.T) {
	client := &fakeSES{}
	sender := NewSESSender(client, "no-reply@example.test")

	if err := sender.Send(context.Background(), EmailMessage{
		To:       "paciente@example.test",
		Subject:  "Sólo texto",
		TextBody: "Este correo lleva un enlace y nada más.",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	body := client.input.Content.Simple.Body
	if body.Html != nil {
		t.Fatalf("un mensaje sin HTML no puede llevar una parte HTML: %+v", body.Html)
	}
	if body.Text == nil || !strings.Contains(*body.Text.Data, "un enlace") {
		t.Fatalf("unexpected text part: %+v", body.Text)
	}

	// Y al revés: el que sí trae HTML lo lleva.
	client = &fakeSES{}
	sender = NewSESSender(client, "no-reply@example.test")
	if err := sender.Send(context.Background(), EmailMessage{
		To: "paciente@example.test", Subject: "Con las dos",
		TextBody: "texto", HTMLBody: "<p>html</p>",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if client.input.Content.Simple.Body.Html == nil {
		t.Fatal("el HTML declarado tiene que viajar")
	}
}

func TestSESSenderRefusesAMessageWithNothingToSay(t *testing.T) {
	client := &fakeSES{}
	sender := NewSESSender(client, "no-reply@example.test")

	if err := sender.Send(context.Background(), EmailMessage{
		To: "paciente@example.test", Subject: "Vacío",
	}); err == nil {
		t.Fatal("un mensaje sin cuerpo tiene que rechazarse")
	}
	if client.calls != 0 {
		t.Fatal("no se puede llamar a SES con un mensaje vacío")
	}
}

func TestSESSenderRequiresARecipient(t *testing.T) {
	client := &fakeSES{}
	sender := NewSESSender(client, "no-reply@example.test")

	if err := sender.Send(context.Background(), EmailMessage{Subject: "s", TextBody: "b"}); err == nil {
		t.Fatal("un mensaje sin destinatario tiene que rechazarse")
	}
	if client.calls != 0 {
		t.Fatal("no se puede llamar a SES sin destinatario")
	}
}
