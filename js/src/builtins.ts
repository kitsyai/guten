import { Engine, PartHTML, PartSubject, PartText, type Template } from "./guten.js";

/**
 * basicNotification is guten's brand-neutral, batteries-included template. It is
 * byte-identical to the Go builtin (go/builtins.go) so both runtimes render the
 * same output. Every brand/visual choice is supplied as data.
 */
export function basicNotification(): Template {
  return {
    name: "basic_notification",
    parts: {
      [PartSubject]: `{{ subject | default: title }}`,
      [PartText]: `Hi {{ name | default: "there" }},

{{ body }}
{% if action_url %}
{{ action_label | default: "Open" }}: {{ action_url }}
{% endif %}
{{ brand_name }}`,
      [PartHTML]: `<!doctype html>
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
  };
}

/** builtins returns guten's batteries-included templates. */
export function builtins(): Template[] {
  return [basicNotification()];
}

/** newWithBuiltins returns an Engine pre-loaded with builtins(). */
export function newWithBuiltins(): Engine {
  const e = new Engine();
  for (const t of builtins()) e.register(t);
  return e;
}
