# Infrastructure Provider Decision Document

**Context:** Non-profit organization. Expected email volume: <100 emails/day (~3,000/month).
Privacy is a priority for both domain and VPS selection.

---

## Email Providers

| Provider | Free Tier | Paid (cheapest) | Notes |
|---|---|---|---|
| **Brevo** (fmr. Sendinblue) | 300/day, 9,000/month | $9/mo (20k emails) | Most generous free tier; EU-based |
| **Resend** | 100/day, 3,000/month | $20/mo (50k emails) | Developer-friendly; clean API |
| **Mailtrap** | 1,000/month | $15/mo (10k emails) | Free tier likely insufficient; good deliverability |
| **SendGrid** | 100/day, 3,000/month | $19.95/mo (50k emails) | Owned by Twilio; free tier just meets our needs |
| **Mailgun** | 100 emails/day (trial only) | ~$0.80/1k emails (Flex) | No permanent free tier; pay-as-you-go ~$2-3/mo |
| **Amazon SES** | 3,000/month (while in EC2) | $0.10/1k emails | Cheapest at scale; complex setup; requires AWS account |
| **Postmark** | 100/month | $15/mo (10k emails) | Excellent deliverability; free tier too small |

### Recommendation: **Brevo (Free tier)**

Brevo's free tier (9,000 emails/month) comfortably covers our <3,000/month usage with a 3x
buffer. It's EU-based (GDPR-compliant), has a solid reputation for deliverability, and includes
basic analytics. No cost until we significantly exceed current projections.

**Runner-up: Resend** — more developer-friendly API, but the free tier is exactly at our limit
(100/day) with no headroom. Upgrade to $20/mo if deliverability or headroom becomes an issue.

---

## Domain Providers

Privacy-respecting providers offer free WHOIS privacy (hiding registrant details from public
lookup) without extra fees.

| Provider | .com/yr | WHOIS Privacy | Jurisdiction | Notes |
|---|---|---|---|---|
| **Cloudflare Registrar** | ~$10-11 | Free | US | At-cost pricing; no markup; requires Cloudflare account |
| **Porkbun** | ~$10-11 | Free | US | Competitive pricing; clean interface |
| **Namecheap** | ~$9-14 | Free (WhoisGuard) | US | Long-standing reputation; occasional upsells |
| **Njalla** | ~$15-36+ | Owned by Njalla | Sweden/Seychelles | Strongest privacy: they own domain on your behalf; higher cost |
| **Gandi** | ~$20 | Free | France | EU-based; pricier; no-ads policy; good reputation |
| **OrangeWebsite** | ~$15 | Free | Iceland | Privacy-focused; green energy hosting |

### Recommendation: **Cloudflare Registrar**

At-cost pricing (~$10-11/yr for `.com`) with free WHOIS privacy and seamless integration with
Cloudflare's free DNS/CDN tier. For a non-profit, the lack of markup is meaningful over time.

**If stronger anonymity is needed: Njalla** — they register the domain in their name on your
behalf, making it harder to associate the domain with your organization. Costs ~$15-36+/yr
depending on TLD. Worth considering if organizational privacy is a concern beyond just WHOIS.

---

## VPS Providers

Privacy-respecting providers: EU/Iceland/Swiss jurisdiction, transparent data practices, ideally
no US cloud act exposure.

| Provider | Entry Plan | Price/mo | Jurisdiction | Notes |
|---|---|---|---|---|
| **Hetzner** | CX22 (2 vCPU, 4GB RAM, 40GB SSD) | ~€3.79 (~$4) | Germany/Finland | Best value; ISO 27001; GDPR; very popular with privacy community |
| **OVHcloud** | Starter (2 vCPU, 2GB RAM, 20GB SSD) | ~$3.50-6 | France/Canada | Large EU provider; competitive pricing; not cutting-edge UX |
| **Infomaniak** | Smallest VPS (2 vCPU, 2GB RAM) | ~CHF 6 (~$7) | Switzerland | Swiss privacy laws; green energy; ethical tech company |
| **Njalla VPS** | 15 USD/mo (1 vCPU, 1GB RAM) | $15 | Sweden/Seychelles | Strong privacy; expensive for specs; Tor support |
| **FlokiNET** | Basic (1 vCPU, 1GB RAM) | ~$4-7 | Iceland/Romania | Privacy-focused; DMCA-ignored; Iceland strong legal protections |
| **DigitalOcean** | Droplet (1 vCPU, 1GB RAM) | $6 | US | Easy to use; US jurisdiction; less privacy-forward |
| **Vultr** | Cloud Compute (1 vCPU, 1GB RAM) | $6 | US | US jurisdiction; similar to DO |

### Recommendation: **Hetzner (CX22)**

At ~$4/month, Hetzner's CX22 offers exceptional value: 2 vCPU, 4GB RAM, 40GB SSD — more than
enough to run this application. Germany-based with GDPR compliance, no US Cloud Act exposure, and
a strong reputation in the privacy and self-hosting community. ISO 27001 certified.

**Runner-up: Infomaniak** — Swiss jurisdiction provides stronger legal privacy protections, ethical
company mission, but costs ~$7/mo for comparable specs. Worth it if Swiss jurisdiction is
specifically desired over German.

---

## Combined Monthly Cost Summary

| Scenario | Email | Domain (amortized) | VPS | **Total/mo** |
|---|---|---|---|---|
| **Recommended (Lean)** | Brevo Free — $0 | Cloudflare ~$0.92 | Hetzner CX22 ~$4 | **~$5/mo** |
| **Privacy-Hardened** | Brevo Free — $0 | Njalla ~$2.50 | Infomaniak ~$7 | **~$9.50/mo** |
| **Maximum Privacy** | Brevo Free — $0 | Njalla ~$2.50 | Njalla VPS ~$15 | **~$17.50/mo** |
| **If email free tier exhausted** | Resend $20 | Cloudflare ~$0.92 | Hetzner CX22 ~$4 | **~$25/mo** |

---

## Decision

| Category | Selected | Monthly Cost |
|---|---|---|
| Email | Brevo (free tier) | $0 |
| Domain | Cloudflare Registrar | ~$0.92 |
| VPS | Hetzner CX22 | ~$4.00 |
| **Total** | | **~$5/mo** |

Start with the **Recommended (Lean)** stack at ~$5/month. If organizational privacy concerns
grow or jurisdiction becomes important, the **Privacy-Hardened** option at ~$9.50/month is a
low-friction upgrade path — only the domain and VPS providers change, not the email setup.
