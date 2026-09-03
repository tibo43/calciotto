// "Share on WhatsApp" for a scheduled match's sign-up panel. There is no
// WhatsApp Business integration here, and deliberately so: the goal is
// posting into a WhatsApp *group*, which the official Cloud API cannot do at
// all (it only supports 1:1 business/customer messaging) — the only
// programmatic alternative is an unofficial client posing as a logged-in
// device, which risks the number it runs on getting banned. wa.me needs
// neither: it's a plain, WhatsApp-published link format that opens WhatsApp
// (web or app) with a message pre-filled, and with no phone number in the
// link, WhatsApp itself prompts the user to pick who to send it to — any
// contact or group they're already in. The click and the final "send" stay
// human, which is exactly what keeps this reliable with zero infrastructure.

// Kept separate from buildWhatsAppShareUrl so a test can assert on the
// readable message without also asserting on its URL-encoding.
//
// The URL is on its own line, in full, rather than hidden behind "Join
// Calciotto" as anchor text: WhatsApp's text messages have no such thing —
// only *bold*/_italic_/~strikethrough~ markdown, no link-with-custom-text
// syntax — so a raw URL is the only thing WhatsApp will actually turn into a
// tappable link.
export function buildWhatsAppShareText({ kickoffLabel, matchUrl }) {
  return `Match: ${kickoffLabel}\nSign-ups are open — Join Calciotto:\n${matchUrl}`;
}

export function buildWhatsAppShareUrl(text) {
  return `https://wa.me/?text=${encodeURIComponent(text)}`;
}
