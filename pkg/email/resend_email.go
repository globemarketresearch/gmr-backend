package email

import (
	"fmt"

	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/domain/order"
	resend "github.com/resendlabs/resend-go"
)

type resendEmailService struct {
	client *resend.Client
	cfg    *config.EmailConfig
}

// NewResendEmailService creates an EmailService backed by the Resend HTTP API.
// It is a drop-in replacement for NewSMTPEmailService.
func NewResendEmailService(cfg *config.EmailConfig) EmailService {
	return &resendEmailService{
		client: resend.NewClient(cfg.ResendAPIKey),
		cfg:    cfg,
	}
}

// sendOne is a small helper that builds and dispatches a single Resend email.
func (r *resendEmailService) sendOne(to, subject, html string, cc []string) error {
	params := &resend.SendEmailRequest{
		From:    r.cfg.From,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}
	if len(cc) > 0 {
		params.Cc = cc
	}
	_, err := r.client.Emails.Send(params)
	return err
}

// ccList returns a non-nil CC slice only when the config has a CC address.
func (r *resendEmailService) ccList() []string {
	if r.cfg.CC != "" {
		return []string{r.cfg.CC}
	}
	return nil
}

// SendFormNotification sends an HTML email notification for a form submission.
// Sends two emails: one admin notification and one client confirmation.
func (r *resendEmailService) SendFormNotification(submission *form.FormSubmission) error {
	// 1. Admin notification
	adminSubject, adminBody := buildEmail(submission, r.cfg.ClientURL)
	if err := r.sendOne(r.cfg.NotifyTo, adminSubject, adminBody, r.ccList()); err != nil {
		return fmt.Errorf("resend admin notification: %w", err)
	}

	// 2. Client confirmation
	clientEmail := strVal(submission.Data["email"])
	if clientEmail == "" {
		return nil
	}

	clientSubject, clientBody := buildClientConfirmationEmail(submission)
	if err := r.sendOne(clientEmail, clientSubject, clientBody, r.ccList()); err != nil {
		return fmt.Errorf("resend client confirmation: %w", err)
	}

	return nil
}

// SendOrderConfirmation sends an HTML order confirmation email to the customer.
func (r *resendEmailService) SendOrderConfirmation(o *order.Order) error {
	subject := fmt.Sprintf("Your Order Confirmation — %s", o.ReportTitle)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Order Confirmation</h2>
  <p>Dear %s,</p>
  <p>Thank you for your purchase! Your order has been received and is being processed.</p>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin:20px 0">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Order ID</strong></td><td>#%d</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Report</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Amount Paid</strong></td><td>%s %.2f</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date</strong></td><td>%s</td></tr>
  </table>
  <p>Your report will be delivered to <strong>%s</strong> within <strong>2–3 business days</strong>.</p>
  <p>If you have any questions, please contact us at <a href="mailto:sales@globemarketresearch.com">sales@globemarketresearch.com</a>.</p>
  <p>Thank you for choosing Globe Market Research!</p>
</body>
</html>`,
		o.CustomerName,
		o.ID,
		o.ReportTitle,
		o.Currency,
		o.Amount,
		o.CreatedAt.Format("January 2, 2006"),
		o.CustomerEmail,
	)

	if err := r.sendOne(o.CustomerEmail, subject, body, r.ccList()); err != nil {
		return fmt.Errorf("resend order confirmation: %w", err)
	}
	return nil
}

// SendOrderAdminNotification sends an HTML notification email to the admin.
func (r *resendEmailService) SendOrderAdminNotification(o *order.Order) error {
	subject := fmt.Sprintf("[NEW ORDER] Payment Received — %s — $%.2f", o.ReportTitle, o.Amount)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#e53935">New Order — Action Required</h2>
  <p>A new order has been placed and payment received. Please deliver the report PDF to the customer within 2–3 business days.</p>
  <h3 style="color:#1a73e8">Customer Details</h3>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin:0 0 20px 0">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Country</strong></td><td>%s</td></tr>
  </table>
  <h3 style="color:#1a73e8">Order Details</h3>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Order ID</strong></td><td>#%d</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Report</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Amount</strong></td><td>%s %.2f</td></tr>
    <tr><td style="background:#f5f5f5"><strong>PayPal Order ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>PayPal Capture ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date</strong></td><td>%s</td></tr>
  </table>
  <p style="margin-top:20px">
    <a href="https://admin.globemarketresearch.com/orders/%d" style="background:#1a73e8;color:#fff;padding:10px 20px;text-decoration:none;border-radius:4px">View Order in Admin</a>
  </p>
</body>
</html>`,
		o.CustomerName,
		o.CustomerEmail,
		o.CustomerCompany,
		o.CustomerPhone,
		o.CustomerCountry,
		o.ID,
		o.ReportTitle,
		o.Currency,
		o.Amount,
		derefStr(o.PaypalOrderID),
		o.PaypalCaptureID,
		o.CreatedAt.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		o.ID,
	)

	if err := r.sendOne(r.cfg.NotifyTo, subject, body, r.ccList()); err != nil {
		return fmt.Errorf("resend order admin notification: %w", err)
	}
	return nil
}
