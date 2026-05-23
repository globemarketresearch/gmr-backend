package email

import (
	"fmt"
	"time"

	"github.com/healthcare-market-research/backend/internal/domain/form"
	"github.com/healthcare-market-research/backend/internal/domain/order"
)

// EmailService defines the interface for sending email notifications.
type EmailService interface {
	SendFormNotification(submission *form.FormSubmission) error
	SendOrderConfirmation(o *order.Order) error
	SendOrderAdminNotification(o *order.Order) error
}

// ── Shared helpers ────────────────────────────────────────────────────────────

func buildClientConfirmationEmail(submission *form.FormSubmission) (subject, body string) {
	fullName := strVal(submission.Data["fullName"])

	switch submission.Category {
	case form.CategoryContact:
		subject = "We received your message — Globe Market Research"
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Thank You for Contacting Us</h2>
  <p>Dear %s,</p>
  <p>We have received your message and our team will get back to you shortly.</p>
  <p>Your reference number is <strong>#%d</strong>. Please keep it for your records.</p>
  <p>If you have any urgent queries, feel free to reach us at <a href="mailto:sales@globemarketresearch.com">sales@globemarketresearch.com</a>.</p>
  <p>Thank you for reaching out to Globe Market Research!</p>
</body>
</html>`, fullName, submission.ID)

	case form.CategoryScheduleDemo:
		preferredTimeLocal := strVal(submission.Data["preferredTimeLocal"])
		userTimezone := strVal(submission.Data["userTimezone"])
		preferredDateTimeUTC := strVal(submission.Data["preferredDateTimeUTC"])

		schedulingRows := ""
		if preferredTimeLocal != "" {
			tzNote := ""
			if userTimezone != "" {
				tzNote = fmt.Sprintf(` <span style="color:#888;font-size:12px">(%s)</span>`, userTimezone)
			}
			schedulingRows += fmt.Sprintf(`<tr><td style="background:#f0f7ff;padding:8px"><strong>Your Preferred Time</strong></td><td style="padding:8px">%s%s</td></tr>`, preferredTimeLocal, tzNote)
		} else if preferredDateTimeUTC != "" {
			if t, err := time.Parse(time.RFC3339, preferredDateTimeUTC); err == nil {
				schedulingRows += fmt.Sprintf(`<tr><td style="background:#f0f7ff;padding:8px"><strong>Preferred Time (UTC)</strong></td><td style="padding:8px">%s</td></tr>`, t.UTC().Format("Mon, Jan 2, 2006 at 3:04 PM UTC"))
			}
		}

		schedulingSection := ""
		if schedulingRows != "" {
			schedulingSection = fmt.Sprintf(`
  <div style="background:#f0f7ff;border:1px solid #bfdbfe;border-radius:8px;padding:16px;margin:20px 0">
    <h3 style="color:#1d4ed8;margin:0 0 12px 0;font-size:15px">Your Requested Schedule</h3>
    <table width="100%%" cellpadding="0" cellspacing="0" style="border-collapse:collapse">
      %s
    </table>
  </div>`, schedulingRows)
		}

		subject = "Demo Request Received — Globe Market Research"
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Your Demo Request is Confirmed!</h2>
  <p>Dear %s,</p>
  <p>Thank you for your interest in Globe Market Research! We have received your demo request and our team will reach out to you within <strong>24 hours</strong> to confirm your session.%s</p>
  <p>Your reference number is <strong>#%d</strong>.</p>
  <p>During the demo, you can expect:</p>
  <ul>
    <li>Live walkthrough of our research platform and reports</li>
    <li>Discussion of your specific research needs</li>
    <li>Overview of subscription options and pricing</li>
    <li>Q&amp;A with our product experts</li>
  </ul>
  <p>If you need to reschedule or have any questions, contact us at <a href="mailto:sales@globemarketresearch.com">sales@globemarketresearch.com</a>.</p>
  <p>We look forward to speaking with you!</p>
</body>
</html>`, fullName, schedulingSection, submission.ID)

	default:
		reportTitle := strVal(submission.Data["reportTitle"])
		subject = "Sample Request Received — Globe Market Research"
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">Sample Request Received</h2>
  <p>Dear %s,</p>
  <p>Thank you for your interest in <strong>%s</strong>. We have received your sample request and will send the sample report to your email within 1–2 business days.</p>
  <p>Your reference number is <strong>#%d</strong>. Please keep it for your records.</p>
  <p>For any questions, contact us at <a href="mailto:sales@globemarketresearch.com">sales@globemarketresearch.com</a>.</p>
  <p>Thank you for choosing Globe Market Research!</p>
</body>
</html>`, fullName, reportTitle, submission.ID)
	}

	return subject, body
}

func buildEmail(submission *form.FormSubmission, clientURL string) (subject, body string) {
	data := submission.Data
	fullName := strVal(data["fullName"])
	submittedAt := submission.CreatedAt.Format(time.RFC1123)
	submissionID := fmt.Sprintf("%d", submission.ID)
	metaRows := buildMetadataRows(submission.Metadata)

	switch submission.Category {
	case form.CategoryContact:
		subject = fmt.Sprintf("[Contact Form] New Submission – %s", fullName)
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">New Contact Form Submission</h2>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Submission ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date / Time</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Full Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Country</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Subject</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5;vertical-align:top"><strong>Message</strong></td><td style="white-space:pre-wrap">%s</td></tr>
    %s
  </table>
</body>
</html>`,
			submissionID, submittedAt, fullName,
			strVal(data["email"]), strVal(data["company"]), strVal(data["country"]),
			strVal(data["phone"]), strVal(data["subject"]), strVal(data["message"]),
			metaRows,
		)

	case form.CategoryScheduleDemo:
		subject = fmt.Sprintf("[Demo Request] New Submission – %s", fullName)

		schedulingRows := ""
		if preferredDateTimeUTC := strVal(data["preferredDateTimeUTC"]); preferredDateTimeUTC != "" {
			if t, err := time.Parse(time.RFC3339, preferredDateTimeUTC); err == nil {
				schedulingRows += fmt.Sprintf(`<tr><td style="background:#e8f4fd"><strong>UTC Time</strong></td><td>%s</td></tr>`, t.UTC().Format("Mon, Jan 2, 2006 at 3:04 PM UTC"))
			}
		}
		if preferredTimeLocal := strVal(data["preferredTimeLocal"]); preferredTimeLocal != "" {
			tz := strVal(data["userTimezone"])
			tzNote := ""
			if tz != "" {
				tzNote = fmt.Sprintf(` <span style="color:#888">(%s)</span>`, tz)
			}
			schedulingRows += fmt.Sprintf(`<tr><td style="background:#e8f4fd"><strong>Client's Local Time</strong></td><td>%s%s</td></tr>`, preferredTimeLocal, tzNote)
		}
		if schedulingRows == "" {
			schedulingRows = `<tr><td colspan="2" style="background:#e8f4fd;color:#888">No preferred date/time selected</td></tr>`
		}

		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#e53935">New Demo Request — Action Required</h2>
  <p>A new demo has been requested. Please contact the client to confirm the session.</p>
  <h3 style="color:#1a73e8;border-bottom:1px solid #e0e0e0;padding-bottom:6px">Scheduling Details</h3>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-bottom:20px">
    %s
  </table>
  <h3 style="color:#1a73e8;border-bottom:1px solid #e0e0e0;padding-bottom:6px">Client Details</h3>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse;margin-bottom:20px">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Submission ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Submitted At</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Full Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td><a href="mailto:%s">%s</a></td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Job Title</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company Size</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Interest Area</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5;vertical-align:top"><strong>Additional Info</strong></td><td style="white-space:pre-wrap">%s</td></tr>
    %s
  </table>
</body>
</html>`,
			schedulingRows,
			submissionID, submittedAt, fullName,
			strVal(data["email"]), strVal(data["email"]),
			strVal(data["company"]), strVal(data["jobTitle"]),
			strVal(data["phone"]), strVal(data["companySize"]),
			strVal(data["interests"]), strVal(data["additionalInfo"]),
			metaRows,
		)

	default:
		subject = fmt.Sprintf("[Request Sample] New Submission – %s", fullName)
		reportTitle := strVal(data["reportTitle"])
		reportSlug := strVal(data["reportSlug"])
		reportTitleCell := reportTitle
		if reportSlug != "" {
			reportTitleCell = fmt.Sprintf(`<a href="%s/reports/%s" style="color:#1a73e8">%s</a>`, clientURL, reportSlug, reportTitle)
		}
		body = fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#1a73e8">New Request Sample Submission</h2>
  <table width="100%%" cellpadding="8" cellspacing="0" style="border-collapse:collapse">
    <tr><td style="background:#f5f5f5;width:160px"><strong>Submission ID</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Date / Time</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Full Name</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Email</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Company</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Job Title</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Country</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Phone</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5"><strong>Report Title</strong></td><td>%s</td></tr>
    <tr><td style="background:#f5f5f5;vertical-align:top"><strong>Additional Info</strong></td><td style="white-space:pre-wrap">%s</td></tr>
    %s
  </table>
</body>
</html>`,
			submissionID, submittedAt, fullName,
			strVal(data["email"]), strVal(data["company"]), strVal(data["jobTitle"]),
			strVal(data["country"]), strVal(data["phone"]),
			reportTitleCell, strVal(data["additionalInfo"]),
			metaRows,
		)
	}

	return subject, body
}

func buildMetadataRows(meta form.SubmissionMetadata) string {
	if meta.IPAddress == "" && meta.PageURL == "" && meta.Referrer == "" {
		return ""
	}
	rows := `<tr><td colspan="2" style="background:#e8eaf6;padding:8px"><strong>Submission Source</strong></td></tr>`
	if meta.IPAddress != "" {
		rows += fmt.Sprintf(`<tr><td style="background:#f0f4ff;width:160px"><strong>IP Address</strong></td><td>%s</td></tr>`, meta.IPAddress)
	}
	if meta.PageURL != "" {
		rows += fmt.Sprintf(`<tr><td style="background:#f0f4ff"><strong>Page URL</strong></td><td style="word-break:break-all">%s</td></tr>`, meta.PageURL)
	}
	if meta.Referrer != "" {
		rows += fmt.Sprintf(`<tr><td style="background:#f0f4ff"><strong>Referrer</strong></td><td style="word-break:break-all">%s</td></tr>`, meta.Referrer)
	}
	return rows
}

// ── Utility helpers ───────────────────────────────────────────────────────────

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
