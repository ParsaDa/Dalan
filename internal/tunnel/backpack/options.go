package backpack

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Kinds of option value, as the panel's form edits them.
const (
	KindBool   = "bool"
	KindInt    = "int"
	KindString = "string"
	KindChoice = "choice"
	KindList   = "list"
)

// Roles an option applies to.
const (
	roleBoth   = "both"
	rolePortal = "portal"
	roleBridge = "bridge"
)

const (
	modeReverse = "reverse"
	modeDirect  = "direct"
)

var (
	hostPattern      = regexp.MustCompile(`^[0-9A-Za-z.:\[\]-]{1,253}$`)
	hostPortPattern  = regexp.MustCompile(`^[0-9A-Za-z.\[\]:-]{1,253}:[0-9]{1,5}$`)
	devicePattern    = regexp.MustCompile(`^[0-9A-Za-z._-]{1,15}$`)
	emailPattern     = regexp.MustCompile(`^[^\s@"\\]{1,64}@[0-9A-Za-z.-]{1,253}$`)
	logLevels        = []string{"trace", "debug", "info", "warn", "error", "fatal"}
	queueDisciplines = []string{"fq_codel", "fq", "pfifo_fast", "cake"}
	// DirectCarriers are the datagram carriers a direct tunnel can run on, as BackPack v1.9.0 spells them.
	DirectCarriers = []string{"udp", "pck", "quic", "sni", "xdi"}
)

// Option is one BackPack setting the panel can write: its key, shape and a hint in both languages.
type Option struct {
	Key     string   `json:"key" example:"edge_ip"`
	Kind    string   `json:"kind" example:"string"`
	Role    string   `json:"role" example:"bridge"`
	Mode    string   `json:"mode" example:"reverse"`
	Group   string   `json:"group" example:"cdn"`
	Choices []string `json:"choices" example:"[\"info\",\"debug\"]"`
	Min     int64    `json:"min" example:"0"`
	Max     int64    `json:"max" example:"0"`
	HintEn  string   `json:"hintEn" example:"Dial this CDN edge address while presenting the tunnel's hostname."`
	HintFa  string   `json:"hintFa" example:"اتصال مستقیم به این آدرس لبهٔ CDN."`

	pattern *regexp.Regexp `json:"-"`
}

// Catalog lists every setting the panel offers, in the order a config file writes them. A key that the
// engine reads but is missing here (the Naive and Xray helpers, the monitor page, spoofed sources) is not offered.
var Catalog = []Option{
	{
		Key: "log_level", Kind: KindChoice, Role: roleBoth, Mode: modeReverse, Group: "general", Choices: logLevels,
		HintEn: "How much the engine logs. info is normal; trace writes a line per connection.",
		HintFa: "میزان لاگ موتور. info معمولی است؛ trace برای هر اتصال یک خط می\u200cنویسد.",
	},
	{
		Key: "bandwidth_mbps", Kind: KindInt, Role: rolePortal, Mode: modeReverse, Group: "limits", Max: 100000,
		HintEn: "Cap on the whole tunnel's speed in Mbit/s. 0 means no cap.",
		HintFa: "سقف سرعت کل تونل بر حسب مگابیت بر ثانیه. ۰ یعنی بدون سقف.",
	},
	{
		Key: "max_connections", Kind: KindInt, Role: rolePortal, Mode: modeReverse, Group: "limits", Max: 1000000,
		HintEn: "Most forwarded connections open at once. 0 means no cap.",
		HintFa: "بیشترین تعداد اتصال هم\u200cزمان. ۰ یعنی بدون سقف.",
	},
	{
		Key: "channel_size", Kind: KindInt, Role: rolePortal, Mode: modeReverse, Group: "limits", Min: 1, Max: 100000,
		HintEn: "How many connections may queue before new ones are dropped. A bigger queue absorbs bursts; it is not faster.",
		HintFa: "تعداد اتصال\u200cهایی که می\u200cتوانند در صف بمانند. صف بزرگ\u200cتر فقط ازدحام را تحمل می\u200cکند و سریع\u200cتر نمی\u200cشود.",
	},
	{
		Key: "nodelay", Kind: KindBool, Role: roleBoth, Mode: modeReverse, Group: "performance",
		HintEn: "Turn off Nagle's algorithm: lower latency, slightly more bandwidth. Good for games and interactive use.",
		HintFa: "الگوریتم Nagle را خاموش می\u200cکند: تأخیر کمتر، مصرف کمی بیشتر. برای بازی و کار تعاملی مناسب است.",
	},
	{
		Key: "keepalive_period", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "performance", Min: 1, Max: 3600,
		HintEn: "Seconds between probes of an idle connection; also how fast a dead peer is noticed.",
		HintFa: "هر چند ثانیه یک\u200cبار اتصال بی\u200cکار بررسی شود؛ سرعت تشخیص قطع\u200cشدن طرف مقابل هم همین است.",
	},
	{
		Key: "heartbeat", Kind: KindInt, Role: rolePortal, Mode: modeReverse, Group: "performance", Min: 1, Max: 3600,
		HintEn: "Seconds between liveness bytes the entry server sends on the control channel.",
		HintFa: "فاصلهٔ ثانیه\u200cای ضربان سلامتی که سرور ورودی روی کانال کنترل می\u200cفرستد.",
	},
	{
		Key: "mss", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "performance", Max: 1460,
		HintEn: "Clamp the TCP segment size. Set it when a tunnel connects and then stalls on the first real transfer.",
		HintFa: "اندازهٔ بسته\u200cهای TCP را محدود می\u200cکند. وقتی تونل وصل می\u200cشود ولی در اولین انتقال واقعی گیر می\u200cکند تنظیمش کنید.",
	},
	{
		Key: "so_rcvbuf", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "performance", Max: 1 << 26,
		HintEn: "Socket receive buffer in bytes. 0 keeps the kernel's own tuning, which is usually right.",
		HintFa: "اندازهٔ بافر دریافت سوکت بر حسب بایت. ۰ یعنی تنظیم خود کرنل که معمولاً درست است.",
	},
	{
		Key: "so_sndbuf", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "performance", Max: 1 << 26,
		HintEn: "Socket send buffer in bytes. 0 keeps the kernel's own tuning.",
		HintFa: "اندازهٔ بافر ارسال سوکت بر حسب بایت. ۰ یعنی تنظیم خود کرنل.",
	},
	{
		Key: "so_pin_tcp", Kind: KindBool, Role: roleBoth, Mode: modeReverse, Group: "performance",
		HintEn: "Pin the TCP socket buffers. Stops kernel auto-tuning and usually costs speed on fast links.",
		HintFa: "بافر سوکت\u200cهای TCP را ثابت می\u200cکند. تنظیم خودکار کرنل را از کار می\u200cاندازد و روی لینک پرسرعت معمولاً کندتر می\u200cشود.",
	},
	{
		Key: "zero_copy", Kind: KindBool, Role: roleBoth, Mode: modeReverse, Group: "performance",
		HintEn: "Let the kernel move bytes between sockets directly. Faster, and the least proven path.",
		HintFa: "کرنل بایت\u200cها را مستقیم بین سوکت\u200cها جابه\u200cجا می\u200cکند. سریع\u200cتر ولی کم\u200cآزموده\u200cترین مسیر است.",
	},
	{
		Key: "skip_optz", Kind: KindBool, Role: roleBoth, Mode: modeReverse, Group: "performance",
		HintEn: "Do not apply the engine's own socket and sysctl tuning at start. For hosts tuned elsewhere.",
		HintFa: "موتور تنظیمات سوکت و sysctl خودش را اعمال نمی\u200cکند. برای سرورهایی که جای دیگری تنظیم شده\u200cاند.",
	},
	{
		Key: "mux_session", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "mux", Min: 1, Max: 256,
		HintEn: "How many multiplexed sessions stay open. Only the mux transports use it.",
		HintFa: "تعداد نشست\u200cهای چندگانه\u200cای که باز می\u200cمانند. فقط ترنسپورت\u200cهای mux از آن استفاده می\u200cکنند.",
	},
	{
		Key: "mux_con", Kind: KindInt, Role: rolePortal, Mode: modeReverse, Group: "mux", Min: 1, Max: 4096,
		HintEn: "How many streams one multiplexed session carries before the next connection waits.",
		HintFa: "هر نشست چندگانه تا چند جریان را حمل کند، قبل از اینکه اتصال بعدی منتظر بماند.",
	},
	{
		Key: "mux_version", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "mux", Min: 1, Max: 2,
		HintEn: "smux protocol version; the two ends negotiate and the lower one wins.",
		HintFa: "نسخهٔ پروتکل smux؛ دو سر مذاکره می\u200cکنند و نسخهٔ کمتر انتخاب می\u200cشود.",
	},
	{
		Key: "mux_framesize", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "mux", Min: 1024, Max: 1 << 20,
		HintEn: "Largest smux frame in bytes.", HintFa: "بزرگ\u200cترین فریم smux بر حسب بایت.",
	},
	{
		Key: "mux_recievebuffer", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "mux", Min: 4096, Max: 1 << 28,
		HintEn: "Per-session receive window in bytes (the key keeps the engine's own spelling).",
		HintFa: "پنجرهٔ دریافت هر نشست بر حسب بایت (املای کلید همان است که خود موتور دارد).",
	},
	{
		Key: "mux_streambuffer", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "mux", Min: 4096, Max: 1 << 28,
		HintEn: "Per-stream receive window in bytes.", HintFa: "پنجرهٔ دریافت هر جریان بر حسب بایت.",
	},
	{
		Key: "fallback_dwell", Kind: KindInt, Role: roleBoth, Mode: modeReverse, Group: "failover", Max: 3600,
		HintEn: "Seconds to hold one fallback transport before trying the next. 0 uses the default.",
		HintFa: "هر ترنسپورت جایگزین چند ثانیه نگه داشته شود قبل از امتحان بعدی. ۰ یعنی پیش\u200cفرض.",
	},
	{
		Key: "fallback_addrs", Kind: KindList, Role: roleBridge, Mode: modeReverse, Group: "failover", pattern: hostPortPattern,
		HintEn: "Extra host:port addresses of the entry server, tried in order when the main one is unreachable.",
		HintFa: "آدرس\u200cهای اضافهٔ host:port سرور ورودی که وقتی آدرس اصلی در دسترس نیست به ترتیب امتحان می\u200cشوند.",
	},
	{
		Key: "health_failover", Kind: KindBool, Role: roleBridge, Mode: modeReverse, Group: "failover",
		HintEn: "Score every address on a timer and keep traffic on the healthiest. Needs more than one address.",
		HintFa: "همهٔ آدرس\u200cها را زمان\u200cبندی\u200cشده امتیاز می\u200cدهد و ترافیک را روی سالم\u200cترین نگه می\u200cدارد. به بیش از یک آدرس نیاز دارد.",
	},
	{
		Key: "load_balance", Kind: KindBool, Role: roleBridge, Mode: modeReverse, Group: "failover",
		HintEn: "Spread data connections over every address. All of them must reach the same entry server.",
		HintFa: "اتصال\u200cهای داده را بین همهٔ آدرس\u200cها پخش می\u200cکند. همه باید به یک سرور ورودی برسند.",
	},
	{
		Key: "edge_ip", Kind: KindString, Role: roleBridge, Mode: modeReverse, Group: "cdn", pattern: hostPattern,
		HintEn: "Dial this CDN edge address while presenting the tunnel's hostname. For the ws and wss transports behind a CDN.",
		HintFa: "مستقیم به این آدرس لبهٔ CDN وصل می\u200cشود ولی نام دامنهٔ تونل را ارائه می\u200cدهد. برای ترنسپورت\u200cهای ws و wss پشت CDN.",
	},
	{
		Key: "simple_auth", Kind: KindBool, Role: roleBoth, Mode: modeReverse, Group: "cdn",
		HintEn: "Authorise a wss tunnel by the raw token. Only for a trusted TLS-terminating proxy in front; set on both ends.",
		HintFa: "تونل wss را با خود توکن تأیید می\u200cکند. فقط پشت پراکسی مورد اعتماد که TLS را خاتمه می\u200cدهد؛ روی هر دو سر بگذارید.",
	},
	{
		Key: "connection_pool", Kind: KindInt, Role: roleBridge, Mode: modeReverse, Group: "connection", Max: 1000,
		HintEn: "Spare connections kept warm so the first request does not wait for a handshake.",
		HintFa: "تعداد اتصال\u200cهای آمادهٔ نگه\u200cداشته\u200cشده تا اولین درخواست منتظر دست\u200cدهی نماند.",
	},
	{
		Key: "aggressive_pool", Kind: KindBool, Role: roleBridge, Mode: modeReverse, Group: "connection",
		HintEn: "Let the pool grow faster under load, holding more idle connections.",
		HintFa: "استخر اتصال زیر بار سریع\u200cتر بزرگ می\u200cشود و اتصال بی\u200cکار بیشتری نگه می\u200cدارد.",
	},
	{
		Key: "dial_timeout", Kind: KindInt, Role: roleBridge, Mode: modeReverse, Group: "connection", Min: 1, Max: 300,
		HintEn: "Seconds one connection attempt may take; also what a filtered address costs before the next is tried.",
		HintFa: "هر تلاش اتصال تا چند ثانیه وقت دارد؛ هزینهٔ یک آدرس فیلترشده هم قبل از رفتن به بعدی همین است.",
	},
	{
		Key: "retry_interval", Kind: KindInt, Role: roleBridge, Mode: modeReverse, Group: "connection", Min: 1, Max: 3600,
		HintEn: "Seconds to wait before dialling again after a failed attempt.",
		HintFa: "بعد از یک تلاش ناموفق چند ثانیه صبر شود.",
	},
	{
		Key: "local_addr", Kind: KindString, Role: roleBridge, Mode: modeReverse, Group: "connection", pattern: hostPattern,
		HintEn: "Source address for the tunnel's connections, to pick the uplink on a multi-homed server.",
		HintFa: "آدرس مبدأ اتصال\u200cهای تونل؛ برای انتخاب خروجی روی سرور با چند کارت شبکه.",
	},
	{
		Key: "interface", Kind: KindString, Role: roleBridge, Mode: modeReverse, Group: "connection", pattern: devicePattern,
		HintEn: "Pin the connections to a named network device. Linux, needs CAP_NET_RAW.",
		HintFa: "اتصال\u200cها را به یک کارت شبکهٔ مشخص می\u200cبندد. فقط لینوکس و نیازمند CAP_NET_RAW.",
	},
	{
		Key: "so_mark", Kind: KindInt, Role: roleBridge, Mode: modeReverse, Group: "connection", Max: math.MaxInt32,
		HintEn: "Stamp an fwmark on the packets, so `ip rule` can give the tunnel its own routing table.",
		HintFa: "روی بسته\u200cها fwmark می\u200cزند تا با `ip rule` برای تونل جدول مسیریابی جدا بسازید.",
	},
	{
		Key: "proxy", Kind: KindString, Role: roleBridge, Mode: modeReverse, Group: "connection",
		HintEn: "Reach the entry server through a proxy, such as socks5://127.0.0.1:1080.",
		HintFa: "اتصال به سرور ورودی را از پراکسی می\u200cگذراند، مثل socks5://127.0.0.1:1080.",
	},
	{
		Key: "proxy_protocol", Kind: KindBool, Role: rolePortal, Mode: modeReverse, Group: "server",
		HintEn: "Prefix each forwarded connection with a PROXY protocol header so the service sees the real client address.",
		HintFa: "قبل از هر اتصال هدر PROXY protocol می\u200cفرستد تا سرویس مقصد آدرس واقعی کاربر را ببیند.",
	},
	{
		Key: "acme_domain", Kind: KindString, Role: rolePortal, Mode: modeReverse, Group: "server", pattern: hostPattern,
		HintEn: "Use a Let's Encrypt certificate for this domain on wss and wssmux instead of a self-signed one.",
		HintFa: "در wss و wssmux به\u200cجای گواهی خودامضا از Let's Encrypt برای این دامنه استفاده می\u200cکند.",
	},
	{
		Key: "acme_email", Kind: KindString, Role: rolePortal, Mode: modeReverse, Group: "server", pattern: emailPattern,
		HintEn: "Where Let's Encrypt sends expiry warnings.", HintFa: "ایمیلی که Let's Encrypt هشدار انقضا را به آن می\u200cفرستد.",
	},

	{
		Key: "carrier", Kind: KindChoice, Role: roleBoth, Mode: modeDirect, Group: "carrier", Choices: DirectCarriers,
		HintEn: "What carries the link: udp is plain; pck sends raw TCP segments; quic looks like HTTP/3; sni adds a TLS hello; xdi hides in ICMP. Both ends must match.",
		HintFa: "چه چیزی لینک را حمل می\u200cکند: udp ساده است؛ pck بسته\u200cهای خام TCP؛ quic شبیه HTTP/3؛ sni یک TLS hello اضافه می\u200cکند؛ xdi داخل ICMP پنهان می\u200cشود. دو سر باید یکی باشند.",
	},
	{
		Key: "local_ip", Kind: KindString, Role: roleBoth, Mode: modeDirect, Group: "carrier", pattern: hostPattern,
		HintEn: "This end's address on the link, with its prefix (set by the panel).",
		HintFa: "آدرس این سر روی لینک همراه با پیشوند (پنل خودش تنظیم می\u200cکند).",
	},
	{
		Key: "peer_ip", Kind: KindString, Role: roleBoth, Mode: modeDirect, Group: "carrier", pattern: hostPattern,
		HintEn: "The other end's address on the link (set by the panel).",
		HintFa: "آدرس سر مقابل روی لینک (پنل خودش تنظیم می\u200cکند).",
	},
	{
		Key: "sni_domain", Kind: KindString, Role: roleBoth, Mode: modeDirect, Group: "carrier", pattern: hostPattern,
		HintEn: "The server name the sni carrier puts in its TLS hello. Pick a domain the path already lets through.",
		HintFa: "نام سروری که حامل sni در TLS hello می\u200cگذارد. دامنه\u200cای انتخاب کنید که مسیر از قبل از آن رد می\u200cکند.",
	},
	{
		Key: "paths", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "carrier", Min: 1, Max: 16,
		HintEn: "Spread the udp carrier over this many consecutive ports, for paths that rate-limit one flow. Same on both ends; the ports must be open.",
		HintFa: "حامل udp را روی این تعداد پورت پشت\u200cسرهم پخش می\u200cکند، برای مسیرهایی که هر جریان را محدود می\u200cکنند. هر دو سر یکی باشد و پورت\u200cها باز باشند.",
	},
	{
		Key: "gre_key", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "carrier", Max: math.MaxUint32,
		HintEn: "GRE key, so more than one logical tunnel can share a carrier. 0 omits it.",
		HintFa: "کلید GRE تا چند تونل منطقی بتوانند یک حامل را به اشتراک بگذارند. ۰ یعنی بدون کلید.",
	},
	{
		Key: "fec_data", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "quality", Max: 64,
		HintEn: "Error correction: for every this-many packets, fec_parity extra ones are sent. Both ends must match; 0 turns it off.",
		HintFa: "تصحیح خطا: برای هر این\u200cتعداد بسته، fec_parity بستهٔ اضافه فرستاده می\u200cشود. دو سر باید یکی باشد؛ ۰ یعنی خاموش.",
	},
	{
		Key: "fec_parity", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "quality", Max: 64,
		HintEn: "Parity packets per group of fec_data. Worth it on a path that drops packets steadily.",
		HintFa: "تعداد بسته\u200cهای توازن برای هر گروه fec_data. برای مسیری که پیوسته بسته گم می\u200cکند می\u200cارزد.",
	},
	{
		Key: "mtu", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "quality", Max: 9000,
		HintEn: "Interface MTU. 0 takes a conservative default.",
		HintFa: "MTU کارت شبکهٔ مجازی. ۰ یعنی مقدار پیش\u200cفرض محتاطانه.",
	},
	{
		Key: "auto_mtu", Kind: KindBool, Role: roleBoth, Mode: modeDirect, Group: "quality",
		HintEn: "Measure what the path really carries once the tunnel is up and set the MTU to match.",
		HintFa: "بعد از بالا آمدن تونل می\u200cسنجد مسیر واقعاً چقدر می\u200cبرد و MTU را همان می\u200cگذارد.",
	},
	{
		Key: "mss_clamp", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "quality", Max: 1460,
		HintEn: "Cap the TCP segment size of connections crossing the link; the fix when transfers stall.",
		HintFa: "اندازهٔ بستهٔ TCP اتصال\u200cهایی که از لینک می\u200cگذرند را محدود می\u200cکند؛ راه\u200cحل وقتی انتقال\u200cها گیر می\u200cکنند.",
	},
	{
		Key: "qdisc", Kind: KindChoice, Role: roleBoth, Mode: modeDirect, Group: "quality", Choices: queueDisciplines,
		HintEn: "Queueing discipline of the interface; it decides jitter. fq_codel is the default.",
		HintFa: "نوع صف کارت شبکهٔ مجازی؛ تعیین\u200cکنندهٔ نوسان تأخیر است. پیش\u200cفرض fq_codel.",
	},
	{
		Key: "txqueuelen", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "quality", Max: 100000,
		HintEn: "Packets the kernel may queue for the interface. Deeper is more latency, not more speed.",
		HintFa: "تعداد بسته\u200cهایی که کرنل برای کارت می\u200cتواند صف کند. صف عمیق\u200cتر یعنی تأخیر بیشتر، نه سرعت بیشتر.",
	},
	{
		Key: "sockbuf", Kind: KindInt, Role: roleBoth, Mode: modeDirect, Group: "quality", Max: 1 << 26,
		HintEn: "Carrier socket buffers in bytes. 0 takes 4 MiB.",
		HintFa: "بافر سوکت\u200cهای حامل بر حسب بایت. ۰ یعنی ۴ مگابایت.",
	},
	{
		Key: "bandwidth_mbps", Kind: KindInt, Role: rolePortal, Mode: modeDirect, Group: "limits", Max: 100000,
		HintEn: "Cap on the forwarded ports' total speed in Mbit/s. 0 means no cap.",
		HintFa: "سقف سرعت کل پورت\u200cهای فوروارد\u200cشده بر حسب مگابیت بر ثانیه. ۰ یعنی بدون سقف.",
	},
	{
		Key: "max_connections", Kind: KindInt, Role: rolePortal, Mode: modeDirect, Group: "limits", Max: 1000000,
		HintEn: "Most forwarded connections open at once. 0 means no cap.",
		HintFa: "بیشترین تعداد اتصال هم\u200cزمان پورت\u200cهای فوروارد\u200cشده. ۰ یعنی بدون سقف.",
	},
}

// OptionsFor lists the options that apply to a tunnel mode, in file order.
func OptionsFor(mode string) []Option {
	out := make([]Option, 0, len(Catalog))
	for _, o := range Catalog {
		if o.Mode == mode {
			out = append(out, o)
		}
	}
	return out
}

func findOption(mode, key string) (Option, bool) {
	for _, o := range Catalog {
		if o.Mode == mode && o.Key == key {
			return o, true
		}
	}
	return Option{}, false
}

func appliesTo(o Option, role string) bool { return o.Role == roleBoth || o.Role == role }

// ValidateOptions checks every value against the catalog; a key the engine does not read, or reads only in the
// other mode, is an error.
func ValidateOptions(mode string, options map[string]any) error {
	for key, value := range options {
		o, ok := findOption(mode, key)
		if !ok {
			return fmt.Errorf("option %q is not offered for a %s tunnel", key, mode)
		}
		if _, err := renderValue(o, value); err != nil {
			return fmt.Errorf("option %q: %w", key, err)
		}
	}
	return nil
}

func wholeNumber(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64
	case int:
		return int64(v), true
	case int64:
		return v, true
	}
	return 0, false
}

func renderValue(o Option, value any) (string, error) {
	switch o.Kind {
	case KindBool:
		b, ok := value.(bool)
		if !ok {
			return "", errors.New("must be true or false")
		}
		return strconv.FormatBool(b), nil
	case KindInt:
		n, ok := wholeNumber(value)
		if !ok {
			return "", errors.New("must be a whole number")
		}
		if n < o.Min || n > o.Max {
			return "", fmt.Errorf("must be between %d and %d", o.Min, o.Max)
		}
		return strconv.FormatInt(n, 10), nil
	case KindChoice:
		s, ok := value.(string)
		if !ok || !slices.Contains(o.Choices, s) {
			return "", fmt.Errorf("must be one of %s", strings.Join(o.Choices, ", "))
		}
		return strconv.Quote(s), nil
	case KindString:
		s, ok := value.(string)
		if !ok {
			return "", errors.New("must be text")
		}
		if err := checkText(o, s); err != nil {
			return "", err
		}
		return strconv.Quote(s), nil
	case KindList:
		items, ok := value.([]any)
		if !ok {
			if typed, isStrings := value.([]string); isStrings {
				for _, s := range typed {
					items = append(items, s)
				}
				ok = true
			}
		}
		if !ok || len(items) == 0 {
			return "", errors.New("must be a list with at least one entry")
		}
		quoted := make([]string, len(items))
		for i, item := range items {
			s, isText := item.(string)
			if !isText {
				return "", errors.New("every entry must be text")
			}
			if err := checkText(o, s); err != nil {
				return "", err
			}
			quoted[i] = strconv.Quote(s)
		}
		return "[" + strings.Join(quoted, ", ") + "]", nil
	}
	return "", fmt.Errorf("unknown option kind %q", o.Kind)
}

// checkText keeps a value to the shape its option takes, which also keeps it from ending the quoted TOML string.
func checkText(o Option, s string) error {
	if o.Key == "proxy" {
		u, err := url.Parse(s)
		if err != nil || (u.Scheme != "socks5" && u.Scheme != "http") || u.Host == "" || strings.ContainsAny(s, "\"\\\r\n") {
			return errors.New("must be a socks5:// or http:// URL")
		}
		return nil
	}
	if o.Key == "local_ip" || o.Key == "peer_ip" {
		host := strings.SplitN(s, "/", 2)[0]
		if net.ParseIP(host) == nil || strings.Count(s, "/") > 1 {
			return errors.New("must be an IP address")
		}
		return nil
	}
	if o.pattern == nil || !o.pattern.MatchString(s) {
		return errors.New("is not a valid value for this option")
	}
	return nil
}

// renderOptions writes the options of the tunnel's mode that apply to this role, in catalog order.
func renderOptions(b *strings.Builder, mode, role string, options map[string]any, skip ...string) error {
	if err := ValidateOptions(mode, options); err != nil {
		return err
	}
	for _, o := range OptionsFor(mode) {
		value, present := options[o.Key]
		if !present || !appliesTo(o, role) || slices.Contains(skip, o.Key) {
			continue
		}
		text, _ := renderValue(o, value)
		fmt.Fprintf(b, "%s = %s\n", o.Key, text)
	}
	return nil
}
