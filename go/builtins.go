package guten

// Builtins returns guten's batteries-included templates. They are deliberately
// brand-neutral: every brand/visual choice (brand_name, accent colour, title,
// body, call-to-action) is supplied as data, so no business or brand knowledge
// lives inside guten. A consumer can use these as-is, override them by
// registering a template of the same name, or ignore them and register its own.
func Builtins() []Template {
	return []Template{basicNotification()}
}

// NewWithBuiltins returns an Engine pre-loaded with Builtins().
func NewWithBuiltins() (*Engine, error) {
	e := New()
	for _, t := range Builtins() {
		if err := e.Register(t); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// basicNotification is a minimal, responsive, single-card email/text template
// parameterised entirely by data:
//
//	subject       (string, optional — falls back to title)
//	title         (string)
//	name          (string, optional — falls back to "there")
//	body          (string)
//	brand_name    (string, optional)
//	accent        (string, optional — hex colour, falls back to #10b981)
//	action_url    (string, optional — renders a button when present)
//	action_label  (string, optional — falls back to "Open")
func basicNotification() Template {
	return Template{
		Name: "basic_notification",
		Parts: map[string]string{
			PartSubject: `{{ subject | default: title }}`,
			PartText: `Hi {{ name | default: "there" }},

{{ body }}
{% if action_url %}
{{ action_label | default: "Open" }}: {{ action_url }}
{% endif %}
{{ brand_name }}`,
			PartHTML: `<!doctype html>
<html lang="en"><body style="margin:0;background:#f4f7fb;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f7fb;border-collapse:collapse;">
    <tr><td align="center" style="padding:32px 16px;">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:600px;background:#ffffff;border:1px solid #e2e8f0;border-radius:16px;border-collapse:separate;overflow:hidden;">
        <tr><td style="height:6px;background:{{ accent | default: '#10b981' }};font-size:0;line-height:0;">&nbsp;</td></tr>
        <tr><td style="padding:24px 28px;">
          {% if brand_name %}<div style="font-size:12px;font-weight:700;letter-spacing:.12em;text-transform:uppercase;color:#64748b;">{{ brand_name | escape }}</div>{% endif %}
          <h1 style="margin:8px 0 0;font-size:22px;line-height:1.2;color:#111827;">{{ title | escape }}</h1>
          <p style="margin:16px 0 0;font-size:16px;line-height:1.6;color:#263244;">Hi {{ name | default: "there" | escape }},</p>
          <p style="margin:12px 0 0;font-size:16px;line-height:1.6;color:#263244;">{{ body | escape }}</p>
          {% if action_url %}<p style="margin:24px 0 0;"><a href="{{ action_url | escape }}" style="display:inline-block;background:{{ accent | default: '#10b981' }};color:#ffffff;text-decoration:none;padding:12px 20px;border-radius:10px;font-weight:700;">{{ action_label | default: "Open" | escape }}</a></p>{% endif %}
        </td></tr>
      </table>
    </td></tr>
  </table>
</body></html>`,
		},
	}
}
