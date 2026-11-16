package rakhsh

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"net"
	"regexp"
	"slices"
	"strings"
)

// Kinds of option value, as the panel's form edits them.
const (
	KindBool   = "bool"
	KindInt    = "int"
	KindString = "string"
	KindChoice = "choice"
	KindList   = "list"
	KindLinks  = "links"
)

// Roles an option applies to; in Rakhsh's terms the portal is the entry and the bridge the exit.
const (
	roleBoth   = "both"
	rolePortal = "portal"
)

// Every Rakhsh option works in both modes; the field exists so the panel reads the same shape as BackPack's.
const modeBoth = "both"

// Carriers are the transports Rakhsh's lanes can run over, as its configuration spells them.
var Carriers = []string{
	"tcp", "tls", "ws", "wss", "quic", "quicv2", "kcp",
	"faketcp", "icmp", "spoof", "l2tcp",
	"shabdiz-gre", "shabdiz-gre6", "shabdiz-esp", "shabdiz-ipip", "shabdiz-6to4", "shabdiz-etherip",
	"shabdiz-l2tp", "shabdiz-mpls", "shabdiz-l2", "shabdiz-udp", "shabdiz-tcp",
}

// RawCarriers forge or craft their own packets, so they need root and the peer's address on both ends.
var RawCarriers = []string{
	"faketcp", "icmp", "spoof", "l2tcp",
	"shabdiz-gre", "shabdiz-gre6", "shabdiz-esp", "shabdiz-ipip", "shabdiz-6to4", "shabdiz-etherip",
	"shabdiz-l2tp", "shabdiz-mpls", "shabdiz-l2",
}

var (
	hostPattern     = regexp.MustCompile(`^[0-9A-Za-z.:\[\]-]{1,253}$`)
	hostPortPattern = regexp.MustCompile(`^[0-9A-Za-z.\[\]:-]{1,253}:[0-9]{1,5}$`)
	pathPattern     = regexp.MustCompile(`^[0-9A-Za-z._~/-]{1,200}$`)
	wsPathPattern   = regexp.MustCompile(`^/[0-9A-Za-z._~/-]{0,100}$`)
	secretPattern   = regexp.MustCompile(`^[0-9A-Za-z._~+=-]{1,128}$`)
	emailPattern    = regexp.MustCompile(`^[^\s@"\\]{1,64}@[0-9A-Za-z.-]{1,253}$`)
	poolPattern     = regexp.MustCompile(`^[0-9A-Za-z.:,]{1,990}$`)
	qdiscPattern    = regexp.MustCompile(`^[a-z_]{2,16}$`)
	ratePattern     = regexp.MustCompile(`^[0-9]+(\.[0-9]+)? ?([kKmMgG]?(bit|bps|b)|[kKmMgG]?B)(/s|ps)?$`)
	sizePattern     = regexp.MustCompile(`^[0-9]+(\.[0-9]+)? ?([kKmMgGtT]i?[bB])?$`)
)

// Option is one Rakhsh setting the panel can write: its key, shape and a hint in both languages.
type Option struct {
	Key     string   `json:"key" example:"edge_ip"`
	Kind    string   `json:"kind" example:"string"`
	Role    string   `json:"role" example:"both"`
	Mode    string   `json:"mode" example:"both"`
	Group   string   `json:"group" example:"carrier"`
	Choices []string `json:"choices" example:"[\"bbr\",\"cubic\"]"`
	Min     int64    `json:"min" example:"1"`
	Max     int64    `json:"max" example:"64"`
	HintEn  string   `json:"hintEn" example:"Dial this CDN edge address while presenting the tunnel's hostname."`
	HintFa  string   `json:"hintFa" example:"اتصال مستقیم به این آدرس لبهٔ CDN."`

	pattern *regexp.Regexp `json:"-"`
}

func newOption(key, kind, group, hintEn, hintFa string) Option {
	return Option{Key: key, Kind: kind, Role: roleBoth, Mode: modeBoth, Group: group, HintEn: hintEn, HintFa: hintFa}
}

func boolOption(key, group, hintEn, hintFa string) Option {
	return newOption(key, KindBool, group, hintEn, hintFa)
}

func intOption(key, group string, minimum, maximum int64, hintEn, hintFa string) Option {
	o := newOption(key, KindInt, group, hintEn, hintFa)
	o.Min, o.Max = minimum, maximum
	return o
}

func choiceOption(key, group string, choices []string, hintEn, hintFa string) Option {
	o := newOption(key, KindChoice, group, hintEn, hintFa)
	o.Choices = choices
	return o
}

func textOption(key, group string, pattern *regexp.Regexp, hintEn, hintFa string) Option {
	o := newOption(key, KindString, group, hintEn, hintFa)
	o.pattern = pattern
	return o
}

func listOption(key, group string, pattern *regexp.Regexp, hintEn, hintFa string) Option {
	o := newOption(key, KindList, group, hintEn, hintFa)
	o.pattern = pattern
	return o
}

func (o Option) onlyOn(role string) Option {
	o.Role = role
	return o
}

// linkOverrides are the settings one [[link]] may override; everything else is inherited from the top level.
var linkOverrides = []string{
	"sni", "ws_path", "ws_credential", "decoy", "edge_ip", "tls_fingerprint", "tls_frag", "cert_path", "key_path",
	"acme_domain", "acme_email", "acme_dir", "acme_port", "acme_staging", "quic_mask", "quic_version",
	"reliability", "faketcp_header", "pace_burst", "pace_gap_us", "pace_pps", "pace_auto",
}

// Catalog lists every setting the panel offers, in the order a config file writes them. Rakhsh's own router
// (inbounds, outbounds, routing, DNS, OpenVPN) and its x-ui sync are not offered: Xray does that work in Dalan.
var Catalog = []Option{
	choiceOption("encryption", "identity", []string{"full", "none"},
		"full encrypts every record with AES-256-GCM; none only authenticates the handshake. Use none only for traffic that is already encrypted end to end.",
		"full هر رکورد را با AES-256-GCM رمز می‌کند؛ none فقط هویت را در دست‌دهی بررسی می‌کند و فقط برای ترافیکی است که خودش رمز شده."),
	boolOption("resume", "identity",
		"Keep TCP streams alive when a lane dies: they move to another lane and go on from the last acknowledged byte.",
		"وقتی یک lane می‌میرد اتصال‌های TCP زنده می‌مانند و از آخرین بایت تأییدشده روی lane دیگر ادامه می‌دهند."),
	intOption("resume_grace", "identity", 1, 3600,
		"Seconds a stream waits with no lane before it is closed.",
		"چند ثانیه یک اتصال بدون هیچ lane منتظر می‌ماند تا بسته شود."),

	intOption("lanes", "tunnel", 1, 64,
		"How many parallel lanes the dialling side opens, across all links.",
		"تعداد lane‌های موازی که سمت اتصال‌دهنده باز می‌کند، روی همهٔ لینک‌ها."),
	intOption("max_connections", "tunnel", 1, 1000000,
		"The most concurrent streams the tunnel carries. Unlimited when empty.",
		"بیشترین تعداد اتصال هم‌زمان تونل. اگر خالی باشد بدون سقف است."),
	choiceOption("preset", "tunnel", []string{"balance", "turbo", "aggressive", "throughput"},
		"Sets heartbeat and keepalive together: aggressive 5/15 s, turbo 8/20 s, balance 10/30 s, throughput 15/30 s.",
		"زمان heartbeat و keepalive را با هم تنظیم می‌کند: aggressive ‏۵/۱۵، turbo ‏۸/۲۰، balance ‏۱۰/۳۰، throughput ‏۱۵/۳۰ ثانیه."),
	intOption("heartbeat", "tunnel", 1, 300,
		"Seconds between lane heartbeats; overrides the preset.",
		"فاصلهٔ heartbeat بین lane‌ها به ثانیه؛ روی preset غالب می‌شود."),
	intOption("keepalive", "tunnel", 5, 3600,
		"Seconds a lane socket may sit idle before TCP keepalive starts; overrides the preset. Stream carriers only.",
		"چند ثانیه سوکت بی‌کار بماند تا keepalive شروع شود؛ روی preset غالب می‌شود. فقط حامل‌های stream."),
	intOption("so_rcvbuf", "tunnel", 4096, 1<<30,
		"Receive buffer of the lane sockets, in bytes. The kernel decides when empty.",
		"اندازهٔ بافر دریافت سوکت lane‌ها به بایت. اگر خالی باشد کرنل تصمیم می‌گیرد."),
	intOption("so_sndbuf", "tunnel", 4096, 1<<30,
		"Send buffer of the lane sockets, in bytes. The kernel decides when empty.",
		"اندازهٔ بافر ارسال سوکت lane‌ها به بایت. اگر خالی باشد کرنل تصمیم می‌گیرد."),
	textOption("bind", "tunnel", hostPattern,
		"Source address for outgoing lanes; it must already exist on this server. It is not sent to the other end, so set it on each end that needs one.",
		"آدرس مبدأ برای lane‌های خروجی؛ باید روی این سرور وجود داشته باشد. به سر دیگر فرستاده نمی‌شود، پس روی هر سری که لازم دارد جدا بگذارید."),

	textOption("sni", "carrier", hostPattern,
		"TLS server name and WebSocket Host header for tls, ws and wss; the listening side derives its certificate from it.",
		"نام سرور TLS و هدر Host در WebSocket برای tls و ws و wss؛ سمت شنونده گواهی خود را از آن می‌سازد."),
	textOption("ws_path", "carrier", wsPathPattern,
		"The path the WebSocket upgrade must request. Both ends must match.",
		"مسیری که درخواست ارتقای WebSocket باید بزند. دو سر باید یکی باشند."),
	textOption("ws_credential", "carrier", secretPattern,
		"A shared string the upgrade request must carry, so guessing the path is not enough.",
		"یک رشتهٔ مشترک که درخواست ارتقا باید داشته باشد تا فقط حدس زدن مسیر کافی نباشد."),
	boolOption("decoy", "carrier",
		"Answer a request that is not a lane with an ordinary web response. On by default for ws and wss.",
		"به درخواستی که lane نیست جواب وب معمولی بده. برای ws و wss پیش‌فرض روشن است."),
	textOption("cert_path", "carrier", pathPattern,
		"A real PEM certificate for the listening side, used instead of the self-signed one. Set both cert_path and key_path.",
		"گواهی PEM واقعی برای سمت شنونده به جای گواهی خودامضا. هر دو cert_path و key_path را بگذارید."),
	textOption("key_path", "carrier", pathPattern,
		"The private key that matches cert_path.",
		"کلید خصوصی مربوط به cert_path."),
	textOption("acme_domain", "carrier", hostPattern,
		"Get and renew a Let's Encrypt certificate for this name. It must already resolve to this server. Instead of cert_path and key_path.",
		"برای این نام گواهی Let's Encrypt بگیر و تمدید کن. نام باید از قبل به این سرور اشاره کند. به جای cert_path و key_path."),
	textOption("acme_email", "carrier", emailPattern,
		"Address the certificate authority sends expiry warnings to.",
		"ایمیلی که مرجع صدور گواهی هشدار انقضا را به آن می‌فرستد."),
	textOption("acme_dir", "carrier", pathPattern,
		"Where the account key and certificate are stored.",
		"محل ذخیرهٔ کلید حساب و گواهی."),
	intOption("acme_port", "carrier", 1, 65535,
		"The port that answers the http-01 challenge; it must be reachable on port 80 from the internet.",
		"پورتی که چالش http-01 را جواب می‌دهد؛ باید از اینترنت روی پورت ۸۰ در دسترس باشد."),
	boolOption("acme_staging", "carrier",
		"Use the Let's Encrypt staging authority while testing a new setup.",
		"هنگام آزمودن تنظیمات تازه از سرور آزمایشی Let's Encrypt استفاده کن."),
	choiceOption("tls_fingerprint", "carrier", []string{"chrome", "firefox", "rustls"},
		"Which TLS client hello to imitate. Read by the dialling side only.",
		"کدام ClientHello تقلید شود. فقط سمت اتصال‌دهنده می‌خواند."),
	intOption("tls_frag", "carrier", 1, 2048,
		"Split the first 2048 bytes of the TLS handshake into pieces of this many bytes so the server name does not arrive in one segment.",
		"۲۰۴۸ بایت اول دست‌دهی TLS را به قطعه‌های این اندازه می‌شکند تا نام سرور در یک بسته نرسد."),
	textOption("edge_ip", "carrier", hostPortPattern,
		"Connect to this address, such as a CDN edge in front of the exit, while still presenting sni. Dialling side only.",
		"به این آدرس، مثلاً یک لبهٔ CDN جلوی سرور خروجی، وصل شو ولی sni را نشان بده. فقط سمت اتصال‌دهنده."),
	boolOption("quic_mask", "carrier",
		"XOR every QUIC datagram with a keystream so packets do not look like QUIC. quic and quicv2 only; both ends must set it.",
		"هر بستهٔ QUIC را با یک کلیدجریان XOR می‌کند تا شبیه QUIC نباشد. فقط quic و quicv2 و هر دو سر باید روشن کنند."),
	intOption("quic_version", "carrier", 0, math.MaxUint32,
		"The QUIC version the dialling side offers first. Default 4278190109 (draft-29).",
		"نسخهٔ QUIC که سمت اتصال‌دهنده اول پیشنهاد می‌دهد. پیش‌فرض ۴۲۷۸۱۹۰۱۰۹ (draft-29)."),
	choiceOption("congestion", "carrier", []string{"bbr", "cubic"},
		"The QUIC congestion controller. quic and quicv2 only.",
		"الگوریتم کنترل ازدحام QUIC. فقط quic و quicv2."),

	listOption("backup_connect", "links", hostPortPattern,
		"More addresses for the same carrier, tried in order after the main address. Dialling side only.",
		"آدرس‌های بیشتر برای همان حامل که پس از آدرس اصلی به ترتیب امتحان می‌شوند. فقط سمت اتصال‌دهنده."),
	newOption("links", KindLinks, "links",
		"More carriers on the same tunnel, in order of preference. If one is blocked the lanes move to the next by themselves. Both ends must list the same links.",
		"حامل‌های بیشتر روی همین تونل به ترتیب اولویت. اگر یکی بسته شود lane‌ها خودکار به بعدی می‌روند. هر دو سر باید همین لینک‌ها را داشته باشند."),
	choiceOption("link_policy", "links", []string{"failover", "spread"},
		"failover: every lane prefers the first link that works. spread: lanes are spread across the links.",
		"failover: همهٔ lane‌ها اولین لینک سالم را ترجیح می‌دهند. spread: lane‌ها بین لینک‌ها پخش می‌شوند."),
	intOption("link_timeout", "links", 2, 600,
		"Seconds one attempt on one link may take before the next link is tried.",
		"چند ثانیه یک تلاش روی یک لینک وقت دارد تا لینک بعدی امتحان شود."),
	intOption("link_probe", "links", 0, 3600,
		"Seconds between checks for a better link while on a fallback. 0 never moves back.",
		"فاصلهٔ بررسی لینک بهتر وقتی روی لینک جایگزین هستیم. ۰ یعنی هیچ‌وقت برنگرد."),

	intOption("pace_burst", "pacing", 1, 1024,
		"How many datagrams to send back to back before each gap. Datagram and raw carriers only.",
		"چند بسته پشت سر هم قبل از هر وقفه فرستاده شود. فقط حامل‌های datagram و raw."),
	intOption("pace_gap_us", "pacing", 0, 1000000,
		"The gap between bursts, in microseconds.",
		"وقفه بین دسته‌ها به میکروثانیه."),
	intOption("pace_pps", "pacing", 1, 10000000,
		"A rate in datagrams per second; overrides pace_burst and pace_gap_us.",
		"نرخ به بسته در ثانیه؛ روی pace_burst و pace_gap_us غالب می‌شود."),
	boolOption("pace_auto", "pacing",
		"Let the carrier find its own rate from the loss the peer reports.",
		"بگذار حامل از روی لاسی که طرف مقابل گزارش می‌دهد نرخ خودش را پیدا کند."),

	textOption("peer_ip", "raw", hostPattern,
		"Needed by raw and shabdiz-gre/esp/ipip/6to4/l2 carriers: the address of the server that dials (the exit server in a reverse tunnel, the entry server in a direct one). The dialling end uses the other server's address by itself.",
		"لازم برای حامل‌های raw و shabdiz-gre/esp/ipip/6to4/l2: آدرس سروری که وصل می‌شود (سرور خروجی در تونل reverse، سرور ورودی در direct). سر وصل‌شونده خودش آدرس سرور مقابل را می‌گذارد."),
	choiceOption("reliability", "raw", []string{"quic", "kcp"},
		"What runs over forged packets to recover from loss. The other end must match.",
		"چه چیزی روی بسته‌های ساختگی برای جبران گم‌شدن اجرا شود. سر دیگر باید یکی باشد."),
	choiceOption("faketcp_header", "raw", []string{"bare", "linux"},
		"bare sends a plain 20-byte TCP header; linux adds the options a Linux stack sends. Try linux if the tunnel stalls under load.",
		"bare هدر ساده ۲۰ بایتی می‌فرستد؛ linux گزینه‌های پشتهٔ لینوکس را هم اضافه می‌کند. اگر تونل زیر بار گیر کرد linux را امتحان کنید."),
	intOption("raw_window_mb", "raw", 1, 64,
		"Flow-control window of the QUIC inside a raw carrier, in MiB. Raise it on long or fast paths.",
		"پنجرهٔ کنترل جریان QUIC درون حامل raw به مگابایت. روی مسیرهای طولانی یا پرسرعت بالا ببرید."),
	choiceOption("raw_congestion", "raw", []string{"cubic", "bbr"},
		"Congestion controller of that inner QUIC.",
		"الگوریتم کنترل ازدحام همان QUIC درونی."),
	intOption("raw_lanes", "raw", 1, 8,
		"How many flows a raw carrier opens. Raise it if uploads are slow. The other end must match.",
		"حامل raw چند جریان باز کند. اگر آپلود کند است بالا ببرید. سر دیگر باید یکی باشد."),
	intOption("fec_data", "raw", 0, 64,
		"Forward error correction: payload packets per group. Set together with fec_parity; both ends must match.",
		"تصحیح خطای رو به جلو: تعداد بستهٔ داده در هر گروه. همراه fec_parity تنظیم شود و دو سر یکی باشند."),
	intOption("fec_parity", "raw", 0, 64,
		"Recovery packets per group. They cost bandwidth on every packet, so use them only on a lossy path.",
		"تعداد بستهٔ بازیابی در هر گروه. روی هر بسته پهنای باند مصرف می‌کنند و فقط برای مسیر پرلاس مناسب‌اند."),
	boolOption("link_encrypt", "raw",
		"Encrypt the raw link layer with the session's encryption. On by default; both ends must match.",
		"لایهٔ لینک raw را با رمزنگاری نشست رمز کن. پیش‌فرض روشن؛ دو سر باید یکی باشند."),
	textOption("spoof_src_ip", "raw", hostPattern,
		"The source address the entry server forges; this server's own when empty. The exit server uses it as the address to expect.",
		"آدرس مبدأ جعلی که سرور ورودی می‌زند؛ اگر خالی باشد آدرس خود سرور. سرور خروجی آن را به‌عنوان آدرسِ مورد انتظار به کار می‌برد."),
	textOption("spoof_src_pool", "raw", poolPattern,
		"Comma-separated addresses to rotate through instead of one.",
		"آدرس‌های جدا شده با ویرگول که به جای یکی بین آن‌ها چرخش می‌شود."),
	textOption("spoof_peer_ip", "raw", hostPattern,
		"The peer's address as it appears on the wire, if it differs from peer_ip.",
		"آدرس طرف مقابل آنطور که روی سیم دیده می‌شود، اگر با peer_ip فرق دارد."),
	textOption("spoof_peer_src", "raw", hostPattern,
		"The source address the exit server forges, so the entry server can recognise its packets. The exit server swaps this and spoof_src_ip by itself.",
		"آدرس مبدأ جعلی که سرور خروجی می‌زند تا سرور ورودی بسته‌هایش را بشناسد. سرور خروجی این و spoof_src_ip را خودش جابه‌جا می‌کند."),
	intOption("spoof_sockbuf", "raw", 4096, 1<<30,
		"The raw socket's buffer size in bytes. On a CPU-bound server a larger buffer can be slower.",
		"اندازهٔ بافر سوکت raw به بایت. روی سرور محدود به CPU بافر بزرگ‌تر ممکن است کندتر باشد."),
	boolOption("spoof_ttl_jitter", "raw",
		"Vary the TTL from one packet to the next.",
		"TTL را از هر بسته به بستهٔ بعد تغییر بده."),
	boolOption("spoof_dscp_jitter", "raw",
		"Vary the DSCP field from one packet to the next.",
		"فیلد DSCP را از هر بسته به بستهٔ بعد تغییر بده."),
	intOption("spoof_rotate_secs", "raw", 0, 86400,
		"Seconds between moves to the next address of spoof_src_pool. 0 never rotates.",
		"فاصلهٔ جابه‌جایی به آدرس بعدی spoof_src_pool به ثانیه. ۰ یعنی هرگز."),

	textOption("l3_addr", "layer3", nil,
		"Turns the tunnel into a routed tun device. This is the entry end's address inside the tunnel, such as 10.77.0.1/30; the exit end takes the next address. Needs root.",
		"تونل را به یک دستگاه tun مسیریابی‌شده تبدیل می‌کند. آدرس سر ورودی داخل تونل، مثلاً 10.77.0.1/30؛ سر خروجی آدرس بعدی را می‌گیرد. نیاز به root دارد."),
	intOption("l3_mtu", "layer3", 576, 9000,
		"MTU of the tun device in bytes. Default 1380.",
		"MTU دستگاه tun به بایت. پیش‌فرض ۱۳۸۰."),
	boolOption("l3_auto_mtu", "layer3",
		"Take the MTU from the carrier instead of l3_mtu.",
		"MTU را از حامل بگیر به جای l3_mtu."),
	intOption("mss_clamp", "layer3", 536, 9000,
		"Rewrite the MSS that TCP handshakes inside the tunnel announce. At most the MTU minus 40.",
		"MSS اعلام‌شده در دست‌دهی TCP داخل تونل را بازنویسی می‌کند. حداکثر MTU منهای ۴۰."),
	boolOption("l3_gre", "layer3",
		"Wrap the inner packets in GRE instead of sending them plain.",
		"بسته‌های داخلی را در GRE بپیچ به جای ارسال ساده."),
	intOption("gre_key", "layer3", 0, math.MaxUint32,
		"The GRE key when l3_gre is on. Both ends must match.",
		"کلید GRE وقتی l3_gre روشن است. دو سر باید یکی باشند."),
	textOption("l3_qdisc", "layer3", qdiscPattern,
		"Queueing discipline of the tun device. Default fq.",
		"نوع صف دستگاه tun. پیش‌فرض fq."),
	intOption("l3_txqueuelen", "layer3", 1, 100000,
		"Transmit queue length of the tun device, in packets. Default 1000.",
		"طول صف ارسال دستگاه tun به بسته. پیش‌فرض ۱۰۰۰."),
	intOption("l3_paths", "layer3", 1, 64,
		"Spread inner packets over at most this many lanes. 1 keeps them in order on one lane.",
		"بسته‌های داخلی را روی حداکثر این تعداد lane پخش کن. ۱ آن‌ها را روی یک lane و به ترتیب نگه می‌دارد."),

	textOption("limit", "limits", ratePattern,
		"Speed cap for both directions, such as 200mbit or 5MB/s. Enforced at the entry end.",
		"سقف سرعت برای هر دو جهت، مثل 200mbit یا 5MB/s. در سر ورودی اعمال می‌شود.").onlyOn(rolePortal),
	textOption("limit_up", "limits", ratePattern,
		"Speed cap for the upload direction.",
		"سقف سرعت جهت آپلود.").onlyOn(rolePortal),
	textOption("limit_down", "limits", ratePattern,
		"Speed cap for the download direction.",
		"سقف سرعت جهت دانلود.").onlyOn(rolePortal),
	textOption("limit_per_stream", "limits", ratePattern,
		"Speed cap applied to each connection separately.",
		"سقف سرعت که روی هر اتصال جداگانه اعمال می‌شود.").onlyOn(rolePortal),
	textOption("quota", "limits", sizePattern,
		"Total traffic per period, both directions together, such as 500GB or 2TB.",
		"کل ترافیک در هر دوره، هر دو جهت با هم، مثل 500GB یا 2TB.").onlyOn(rolePortal),
	choiceOption("quota_period", "limits", []string{"day", "week", "month", "never"},
		"When the quota counter resets. Default month.",
		"شمارندهٔ سهمیه کی صفر شود. پیش‌فرض month.").onlyOn(rolePortal),
	textOption("quota_state", "limits", pathPattern,
		"Where the usage counters are saved so a restart does not lose them.",
		"محل ذخیرهٔ شمارنده‌ها تا با راه‌اندازی دوباره از بین نروند.").onlyOn(rolePortal),
}

// OptionsFor lists the options that apply to a tunnel mode, in file order.
func OptionsFor(mode string) []Option {
	if mode != "reverse" && mode != "direct" {
		return []Option{}
	}
	return slices.Clone(Catalog)
}

func findOption(mode, key string) (Option, bool) {
	if mode != "reverse" && mode != "direct" {
		return Option{}, false
	}
	for _, o := range Catalog {
		if o.Key == key {
			return o, true
		}
	}
	return Option{}, false
}

// ValidateOptions checks every value against the catalog; a key the engine does not read is an error.
func ValidateOptions(mode string, options map[string]any) error {
	for _, key := range sortedKeys(options) {
		o, ok := findOption(mode, key)
		if !ok {
			return fmt.Errorf("option %q is not offered for a %s tunnel", key, mode)
		}
		if o.Kind == KindLinks {
			if err := checkLinks(options["links"], options); err != nil {
				return fmt.Errorf("option %q: %w", key, err)
			}
			continue
		}
		if err := checkValue(o, options[key]); err != nil {
			return fmt.Errorf("option %q: %w", key, err)
		}
	}
	if _, hasData := options["fec_data"]; hasData != hasKey(options, "fec_parity") {
		return errors.New(`options "fec_data" and "fec_parity" must be set together`)
	}
	return nil
}

func hasKey(options map[string]any, key string) bool {
	_, ok := options[key]
	return ok
}

func sortedKeys(options map[string]any) []string {
	keys := make([]string, 0, len(options))
	for key := range options {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func wholeNumber(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64
	case int:
		return int64(v), true
	case int64:
		return v, true
	case uint32:
		return int64(v), true
	}
	return 0, false
}

func textItems(value any) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		return v, true
	case []any:
		items := make([]string, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			items[i] = s
		}
		return items, true
	}
	return nil, false
}

// checkValue judges one value by its option's kind; whatever it accepts is safe inside a quoted TOML string.
func checkValue(o Option, value any) error {
	switch o.Kind {
	case KindBool:
		if _, ok := value.(bool); !ok {
			return errors.New("must be true or false")
		}
	case KindInt:
		n, ok := wholeNumber(value)
		if !ok {
			return errors.New("must be a whole number")
		}
		if n < o.Min || n > o.Max {
			return fmt.Errorf("must be between %d and %d", o.Min, o.Max)
		}
	case KindChoice:
		s, ok := value.(string)
		if !ok || !slices.Contains(o.Choices, s) {
			return fmt.Errorf("must be one of %s", strings.Join(o.Choices, ", "))
		}
	case KindString:
		s, ok := value.(string)
		if !ok {
			return errors.New("must be text")
		}
		return checkText(o, s)
	case KindList:
		items, ok := textItems(value)
		if !ok || len(items) == 0 {
			return errors.New("must be a list of text with at least one entry")
		}
		for _, s := range items {
			if err := checkText(o, s); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown option kind %q", o.Kind)
	}
	return nil
}

func checkText(o Option, s string) error {
	if o.Key == "l3_addr" {
		host, _, hasPrefix := strings.Cut(s, "/")
		if net.ParseIP(host) == nil || (hasPrefix && !validPrefix(s)) {
			return errors.New("must be an IP address with an optional prefix length")
		}
		return nil
	}
	if o.pattern == nil || !o.pattern.MatchString(s) {
		return errors.New("is not a valid value for this option")
	}
	return nil
}

func validPrefix(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// checkLinks validates the structured links option: each entry names a known carrier and an address,
// carries only overrides one link may set, and no two entries share a carrier and address.
func checkLinks(value any, options map[string]any) error {
	entries, ok := value.([]any)
	if !ok || len(entries) == 0 {
		return errors.New("must be a list with at least one link")
	}
	seen := map[string]int{}
	for i, raw := range entries {
		n := i + 1
		entry, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("entry %d: must be an object with a carrier and an address", n)
		}
		carrier, _ := entry["carrier"].(string)
		if !slices.Contains(Carriers, carrier) {
			return fmt.Errorf("entry %d: unknown carrier %q", n, carrier)
		}
		addr, _ := entry["addr"].(string)
		if !hostPortPattern.MatchString(addr) {
			return fmt.Errorf("entry %d: addr must be host:port", n)
		}
		if slices.Contains(RawCarriers, carrier) && !hasKey(options, "peer_ip") {
			return fmt.Errorf("entry %d: carrier %q needs the option peer_ip", n, carrier)
		}
		key := carrier + " " + addr
		if first, dup := seen[key]; dup {
			return fmt.Errorf("entries %d and %d use the same carrier and address", first, n)
		}
		seen[key] = n
		for _, field := range sortedKeys(entry) {
			if field == "carrier" || field == "addr" {
				continue
			}
			if !slices.Contains(linkOverrides, field) {
				return fmt.Errorf("entry %d: %q cannot be set on a single link", n, field)
			}
			o, _ := findOption("reverse", field)
			if err := checkValue(o, entry[field]); err != nil {
				return fmt.Errorf("entry %d: %q %w", n, field, err)
			}
		}
	}
	return nil
}

// SharedOptions is what a setup link carries: bind names an address of one server, so it stays behind.
func SharedOptions(options map[string]any) map[string]any {
	shared := maps.Clone(options)
	delete(shared, "bind")
	return shared
}
