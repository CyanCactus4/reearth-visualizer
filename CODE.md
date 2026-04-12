# CODE.md — разбор основных блоков кода

Здесь лежат **настоящие куски кода** из репозитория Re:Earth Visualizer. После каждого куска идёт объяснение **подробно и простым языком**: что именно делает программа по шагам, зачем это нужно в жизни продукта, и как этот кусок стыкуется с соседними. Термины вроде «горутина» или «middleware» по возможности сразу расшифровываются одной фразой.

**Как устроено в целом (без предположений, что вы уже писали бэкенд):** обычный сайт общается с сервером короткими запросами «спросил — получил ответ». Редактору нескольких людей **одновременно** так неудобно: пришлось бы каждую секунду опрашивать сервер «есть новости?». Поэтому включают **WebSocket**: это как телефонная линия, которая **остаётся открытой**, и сервер или браузер могут сказать что-то **в любой момент**. В этом проекте такая линия открывается по адресу вроде **`/api/collab/ws`**, в строке запроса передаётся **`projectId`**. На сервере все, кто указал один и тот же проект, попадают в одну условную **«комнату»** — как чат без истории в начале, но с умом: туда же приходят не только сообщения чата, но и **правки сцены**, **курсоры**, **блокировки** и т.д. Каждое сообщение — это **JSON**. Сначала смотрят на короткое поле **`t`** (тип сообщения). Если там **`apply`**, внутри большого поля **`d`** ещё есть поле **`kind`**: оно говорит, **что именно** поменяли (виджет, слой, стиль, свойство…). Сервер проверяет права, пишет итог в **MongoDB** и рассылает остальным короткое «вот что случилось», чтобы их экраны подтянулись к одной правде.

---

## 1. Хаб коллаборации: комнаты, Redis, лимитеры, часы

Файл `server/internal/collab/hub.go` описывает главный объект **`Hub`**. Можно думать о нём как о **диспетчере чата для редактора**: он знает, кто в какой «комнате» (проекте) сидит, и может переслать сообщение всем в этой комнате.

**Зачем столько полей в структуре:** одна программа-сервер должна одновременно обслуживать много проектов и много вкладок браузера. Поэтому внутри хаба есть:

- список **комнат** (`rooms`): ключ — строка с id проекта, значение — все подключённые к этому проекту клиенты;
- при необходимости **Redis**: если серверов несколько, они через Redis перекидывают сообщения, чтобы пользователи на разных машинах сервера всё равно были в одной «комнате»;
- **блокировки** (`locks`, `lockRedis`): чтобы двое не правили один и тот же виджет без согласия;
- **ограничители частоты** (поля с `Limiter` и `Every`): чтобы один человек не засыпал всех сообщениями чата или курсорами за секунду;
- **номер версии сцены** и подписчики (`sceneRevSubs`): клиенту важно понимать, не устарели ли у него данные;
- **«часы»** для отдельных полей (`widgetClocks`, `propertyFieldClocks` и т.д.): если двое поменяли одно и то же, серверу нужно правило «чья правка победила», не лоча при этом всю сцену на минуту;
- **откат (undo)** и снимки сцены — отдельные поля внизу структуры.

```17:114:server/internal/collab/hub.go
// Hub routes messages between WebSocket clients in the same project room.
// When Redis is configured, messages are also relayed to other server instances.
type Hub struct {
	mu sync.RWMutex

	instanceID string
	rooms      map[string]*room // key: project ID string

	relay *redisRelay

	locks     *lockTable
	lockTTL   time.Duration
	lockRedis *redis.Client // same as relay client when Redis is enabled; distributed locks

	chatMaxRunes int
	chatEvery    time.Duration
	chatLimiters sync.Map // key: projectID + "\x00" + userID -> *rate.Limiter (created lazily)

	cursorEvery    time.Duration
	cursorLimiters sync.Map // projectID + "\x00" + userID

	activityTypingEvery time.Duration
	activityMoveEvery   time.Duration
	activityLimiters    sync.Map // projectID + "\x00" + userID + "\x00" + kind

	chatStore  ChatHistoryStore
	applyAudit ApplyAuditStore

	sceneRevSubMu sync.Mutex
	sceneRevSubs  map[string][]chan int64 // scene ID → subscribers (buffered chans)

	// Per-widget field LWW clocks (in-memory; resets on process restart).
	widgetClockMu sync.Mutex
	widgetClocks  map[string]int64

	// Per-property-field LWW clocks (same Redis client as widget clocks when configured).
	propertyFieldClockMu sync.Mutex
	propertyFieldClocks  map[string]int64

	// Per-property-field HLC (CRDT LWW register timestamps); in-memory when Redis absent.
	propertyFieldHLCMemory *propertyFieldHLCMemory

	// Per-property document clock for merge_property_json (CAS).
	propertyDocClockMu sync.Mutex
	propertyDocClocks  map[string]int64

	// Serializes property-field collab paths (integer LWW + HLC CRDT) vs Mongo apply on this instance.
	propertyCollabApplyMu sync.Mutex

	opStack            CollabOpStack
	sceneSnapshotStore SceneSnapshotStore
	snapMu             sync.Mutex
	snapLastAt         map[string]time.Time // scene ID → last snapshot attempt
	mentionWebhook     string
}

type room struct {
	mu    sync.Mutex
	conns map[*Conn]struct{}
}

func newRoom() *room {
	return &room{conns: make(map[*Conn]struct{})}
}

func NewHub(o Options) *Hub {
	ttl := o.lockTTL()
	h := &Hub{
		instanceID:   uuid.NewString(),
		rooms:        make(map[string]*room),
		locks:        newLockTable(),
		lockTTL:      ttl,
		chatMaxRunes: o.chatMaxRunes(),
		chatEvery:    o.chatMinInterval(),

		cursorEvery:            o.cursorMinInterval(),
		activityTypingEvery:    o.activityTypingInterval(),
		activityMoveEvery:      o.activityMoveInterval(),
		chatStore:              o.ChatHistory,
		applyAudit:             o.ApplyAudit,
		sceneRevSubs:           make(map[string][]chan int64),
		widgetClocks:           make(map[string]int64),
		propertyFieldClocks:    make(map[string]int64),
		propertyFieldHLCMemory: newPropertyFieldHLCMemory(),
		propertyDocClocks:      make(map[string]int64),
		opStack:                o.OpStack,
		sceneSnapshotStore:     o.SceneSnapshot,
		snapLastAt:             make(map[string]time.Time),
		mentionWebhook:         strings.TrimSpace(o.MentionWebhookURL),
	}
	if o.RedisURL != "" {
		if r := newRedisRelay(o.RedisURL, h.instanceID); r != nil {
			h.relay = r
			h.lockRedis = r.Client()
		}
	}
	return h
}
```

**Подробный разбор (можно читать по пунктам):**

1. **`mu sync.RWMutex`** — это «замок» на весь хаб. Много горутин одновременно смотрят, кто в какой комнате. Читать карту комнат можно **многим потокам сразу** (буква **R** в названии — от слова *read*). Менять структуру комнат тяжелее — тогда берётся обычный полный замок. Так сервер не путает данные при большом числе подключений.

2. **`instanceID`** — случайная строка, которая отличает **этот запущенный сервер** от соседнего. Когда сообщение приходит из Redis, сервер смотрит: «это я сам только что отправил, или это пришло с другого сервера?» Так не получается бесконечно гонять одно и то же сообщение по кругу.

3. **`room` и `conns`** — одна комната — это просто **набор соединений**. Почему `map[*Conn]struct{}`, а не список? В Go так часто делают **множество без дубликатов**: соединение либо есть в комнате, либо нет, и удалять его очень дёшево.

4. **`sceneRevSubs`** — кто-то на клиенте может **ждать**, когда номер версии сцены изменится. Вместо того чтобы постоянно опрашивать базу, сервер может положить число в канал. Это похоже на звонок: «сцена обновилась, вот новый номер».

5. **Комментарии в коде про LWW, HLC, document clock** — это разные способы ответить на вопрос: **«если двое изменили одно поле, что оставить?»** Без таких правил последний сохранённый в MongoDB выиграл бы случайно. Здесь правила зашиты в код явно, и они работают **на уровне отдельных полей**, а не «заморозим весь проект на каждую букву».

6. **`NewHub`** в конце вставки: если в настройках указан адрес **Redis**, поднимается связь и для пересылки сообщений, и для **блокировок между серверами**. Если Redis нет — коллаборация всё равно работает, но только **на одном** экземпляре сервера (для локальной разработки этого часто достаточно).

---

## 2. Одно WebSocket-соединение: чтение, запись, отключение

**`Conn`** — это «один участник» в комнате. Обычно это **одна вкладка браузера** одного пользователя. Уже на этом этапе известно: какой проект, какая сцена, кто залогинен (`userID`), какой таб (`clientID`), и есть ли права (`operator`). Всё это кладётся в структуру, чтобы любое следующее сообщение обрабатывалось **в контексте правильного человека и проекта**.

```12:55:server/internal/collab/conn.go
// Conn is one WebSocket client in a project room.
type Conn struct {
	hub       *Hub
	ws        *websocket.Conn
	projectID string
	sceneID   id.SceneID
	userID    string
	clientID  string
	photoURL  string
	operator  *usecase.Operator
	bgCtx     context.Context
	send      chan []byte
}

func (c *Conn) readPump(maxBytes int, lim *rate.Limiter, onMessage func([]byte) error) {
	defer func() {
		c.hub.unregister(c)
		_ = c.ws.Close()
	}()
	c.ws.SetReadLimit(int64(maxBytes))
	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		if lim != nil && !lim.Allow() {
			return
		}
		if err := onMessage(msg); err != nil {
			return
		}
	}
}

func (c *Conn) writePump() {
	defer func() {
		_ = c.ws.Close()
	}()
	for msg := range c.send {
		if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
```

**Подробный разбор (простыми словами):**

**Зачем в `Conn` столько полей.** Когда соединение уже открыто, серверу всё время нужно помнить «кто я и где я». Поэтому там лежит ссылка на **`hub`** (чтобы отписаться при отключении), сам сокет **`ws`**, строка **`projectID`** (в какой комнате сидим), **`sceneID`** (какая сцена у этого проекта), **`userID`** и **`photoURL`** (для чата и списка присутствия), **`operator`** — это объект «текущий пользователь и его права», которым пользуются обычные запросы к базе. Без него нельзя честно записать изменение в Mongo от имени залогиненного человека.

**Что такое `bgCtx` и зачем он отдельно.** Обычный HTTP-запрос живёт недолго: сервер ответил — контекст можно отменить. WebSocket наоборот может висеть **минутами**. Если бы мы использовали только контекст HTTP-ручки, то длинная операция сохранения сцены могла бы **оборваться** просто потому, что «исходный запрос на upgrade уже формально завершён». Поэтому в `ServeWS` делают контекст **без отмены по завершению запроса** (`WithoutCancel`) и кладут его в соединение как `bgCtx`. Идея простая: **пока жив сокет, фоновые действия не должны глушиться таймаутом старого HTTP.**

**Зачем `clientId`.** Один и тот же человек может открыть **две вкладки**. Для системы это почти два разных клиента: у каждой вкладки свой WebSocket. Если не различать вкладки, можно случайно показать уведомление «коллега что-то применил», хотя это **сама вторая вкладка** того же человека, или перепутать блокировку виджета. Строка **`clientId`** — это произвольный id вкладки, который фронт кладёт в URL или в тело сообщения.

**Как работает `readPump` (чтение).** Это бесконечный цикл: прочитать сообщение из сокета → проверить лимит частоты → передать байты в функцию **`onMessage`** (там уже разбор JSON и вызов хаба). Если чтение упало (сеть, закрытие вкладки) или обработчик вернул ошибку — цикл заканчивается. Слово **`defer`** в Go значит: «когда выйдем из функции — **обязательно** выполнить этот код». Там вызывается **`unregister`**: соединение убирается из комнаты, иначе в списке «кто онлайн» остался бы «призрак».

**Два ограничения размера и скорости.** `SetReadLimit` не даёт прислать **одно** сообщение больше заданного размера (защита от полного забивания памяти). `rate.Limiter` не дает слать **слишком часто** много маленьких сообщений (защита от спама и от случайных циклов на клиенте).

**Как работает `writePump` (запись).** Отдельная горутина читает из канала **`send`** и пишет в WebSocket. Зачем канал, а не прямой `WriteMessage` из любого места? Так проще: **много частей кода** могут положить готовый JSON в очередь, а одна горутина пишет в сеть аккуратно по одному сообщению. Если запись в сокет ошиблась — цикл выходит (клиент, скорее всего, ушёл).

**Связь с `enqueueJSON` (ниже по CODE.md / в `apply.go`).** Когда очередь `send` переполнена, сервер **не ждёт** — он отбрасывает лишнее. Это жёстко по отношению к одному медленному клиенту, но **мягко** по отношению ко всем остальным: хаб не встаёт колом из-за одного зависшего браузера.

---

## 3. HTTP → WebSocket: проверки доступа и запуск двух горутин

Здесь начинается «вход в коллаборацию». До открытия WebSocket всё ещё обычный **HTTP**: есть cookies, заголовки, middleware, который положил в контекст пользователя. Функция **`ServeWS`** не «крутит сокет сама по себе» — она **возвращает обработчик** для фреймворка Echo: когда пользователь открывает URL вида `/api/collab/ws?...`, Echo вызывает эту функцию.

```25:125:server/internal/collab/ws.go
// ClientMessage is the minimal supported inbound protocol (v1).
type ClientMessage struct {
	V int             `json:"v"`
	T string          `json:"t"`
	D json.RawMessage `json:"d,omitempty"`
}

// ServeWS returns an Echo handler for GET /api/collab/ws?projectId=...
func ServeWS(hub *Hub, cfg *config.CollabConfig, allowedOrigins []string) echo.HandlerFunc {
	maxBytes := cfg.MaxMessageBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxMessageBytes
	}
	msgsPerSec := cfg.MaxMessagesPerSec
	if msgsPerSec <= 0 {
		msgsPerSec = defaultMsgsPerSec
	}

	up := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, o := range allowedOrigins {
				if o != "" && o == origin {
					return true
				}
			}
			return false
		},
	}

	return func(c echo.Context) error {
		op := adapter.Operator(c.Request().Context())
		if op == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		pidStr := strings.TrimSpace(c.QueryParam("projectId"))
		if pidStr == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "projectId is required")
		}
		pid, err := id.ProjectIDFrom(pidStr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid projectId")
		}

		uc := adapter.Usecases(c.Request().Context())
		if uc == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal error")
		}

		pj, err := uc.Project.FindActiveById(c.Request().Context(), pid, op)
		if err != nil || pj == nil {
			return echo.NewHTTPError(http.StatusForbidden, "project not accessible")
		}
		sceneID, err := resolveProjectSceneForAccess(c.Request().Context(), uc, op, pj, pid)
		if err != nil {
			return err
		}

		ws, err := up.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}

		bgCtx := context.WithoutCancel(c.Request().Context())
		userID := ""
		photoURL := ""
		if u := adapter.User(bgCtx); u != nil {
			userID = u.ID().String()
			if md := u.Metadata(); md != nil {
				photoURL = md.PhotoURL()
			}
		}
		tabClientID := NormalizeCollabClientID(c.QueryParam("clientId"))
		conn := &Conn{
			hub:       hub,
			ws:        ws,
			projectID: pid.String(),
			sceneID:   sceneID,
			userID:    userID,
			clientID:  tabClientID,
			photoURL:  photoURL,
			operator:  op,
			bgCtx:     bgCtx,
			send:      make(chan []byte, sendChannelBuf),
		}
		hub.register(conn)

		lim := rate.NewLimiter(rate.Limit(msgsPerSec), msgsPerSec)

		go conn.writePump()
		conn.readPump(maxBytes, lim, func(raw []byte) error {
			return handleClientMessage(hub, conn, raw, maxBytes)
		})
		return nil
	}
}
```

**Подробный разбор (простыми словами):**

**Что такое `ClientMessage`.** Это «конверт», в который клиент упаковывает **любое** своё сообщение. Поле **`v`** — номер версии протокола (сейчас везде **1**: если завтра поменяют формат, можно будет ввести `v: 2` и не ломать старых клиентов сразу). Поле **`t`** — короткое **слово-тип**: например `ping` или `apply`. Поле **`d`** — это **внутренности** конверта в виде сырого JSON (`json.RawMessage`): сервер сначала смотрит только на `t`, а `d` разбирает уже в другой функции. Так проще и быстрее, чем каждый раз парсить огромную структуру под все случаи.

**Что происходит до открытия сокета (важно для безопасности).** Сначала проверяется: есть ли вообще **залогиненный пользователь** (`Operator`). Потом читается **`projectId`** из строки запроса и проверяется, что это похоже на настоящий id. Потом из контекста достаются **use case**-ы приложения и вызывается проверка вроде «этот пользователь **имеет право** на этот проект?». Если нет — соединение **не открывают** (ошибка 403 и т.п.). Иначе любой мог бы подставить чужой `projectId` и слушать чужую комнату. Дополнительно вызывается **`resolveProjectSceneForAccess`**: нужно понять, с **какой сценой** связан проект, чтобы дальше все apply шли в правильный контекст.

**Зачем `CheckOrigin`.** Браузер при WebSocket шлёт заголовок **Origin** (с какого сайта открыли страницу). Сервер сравнивает его со списком **`allowedOrigins`**. Если Origin не из списка — upgrade **отклоняется**. Это защита от класса атак, когда вредоносный сайт в браузере жертвы пытается открыть сокет **к вашему API** с чужих страниц. Если заголовка Origin нет (не браузер, тест, странный клиент) — в этом коде Origin **пропускают**: иначе ломались бы легитимные сценарии.

**Что происходит сразу после `Upgrade`.** Создаётся объект **`Conn`** со всеми полями (проект, сцена, пользователь, id вкладки, канал `send`). Соединение **регистрируется** в хабе — его добавляют в комнату проекта. Потом запускаются **две** вещи: в фоне (`go`) — **`writePump`** (пишет в сеть из очереди), в текущей горутине — **`readPump`** (читает из сети и зовёт `handleClientMessage`). Почему так: чтение **блокирует** поток — для WebSocket это нормально, соединение и должно «висеть» и ждать сообщения. Запись при этом не мешает чтению, потому что вынесена в отдельную горутину.

**Лимитер `msgsPerSec`.** На каждое соединение создаётся ограничитель: не больше N сообщений в секунду **в среднем**, с небольшим «запасом» (burst). Это как счётчик: если клиент сошёл с ума и шлёт тысячу пакетов в секунду, сервер просто **закроет** чтение и освободит ресурсы.

---

## 4. Диспетчер входящих сообщений по полю `t`

Сообщение уже пришло байтами. Его парсят в структуру **`ClientMessage`**. Дальше стоит обычный **`switch` по полю `t`**: как светофор — в зависимости от типа вызывается разная ветка. Это самый простой способ сделать протокол понятным: одно поле решает, **какая дверь** открывается.

```128:194:server/internal/collab/ws.go
func handleClientMessage(hub *Hub, from *Conn, raw []byte, maxBytes int) error {
	ctx := from.bgCtx
	var m ClientMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.V != 1 {
		return fmt.Errorf("unsupported protocol version")
	}
	switch m.T {
	case "ping":
		resp, _ := json.Marshal(ClientMessage{V: 1, T: "pong"})
		select {
		case from.send <- resp:
		default:
		}
		return nil
	case "relay":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		hub.broadcastFromClient(ctx, from.projectID, raw, from)
		return nil
	case "apply":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("apply requires d")
		}
		return dispatchApply(ctx, hub, from, m.D)
	case "lock":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("lock requires d")
		}
		return dispatchLock(ctx, hub, from, m.D)
	case "chat":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("chat requires d")
		}
		return dispatchChat(ctx, hub, from, m.D)
	case "cursor":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("cursor requires d")
		}
		return dispatchCursor(ctx, hub, from, m.D)
	case "activity":
		if len(raw) > maxBytes {
			return errors.New("message too large")
		}
		if len(m.D) == 0 {
			return fmt.Errorf("activity requires d")
		}
		return dispatchActivity(ctx, hub, from, m.D)
	default:
		return fmt.Errorf("unknown message type")
	}
}
```

**Подробный разбор (простыми словами):**

**Ветка `ping`.** Браузер может периодически слать «ты здесь?», чтобы прокси и сеть не закрыли «тихое» соединение. Сервер отвечает **`pong`**. Ответ кладут в канал **`from.send`** конструкцией `select`: если очередь на запись переполнена, срабатывает ветка **`default`** — ответ **просто не кладут**, вместо того чтобы встать и ждать. Идея: пинг — не критичное сообщение; лучше потерять один `pong`, чем заблокировать весь сервер на этом клиенте.

**Ветка `relay`.** Это режим «**перешли всем как есть**»: сервер почти не лезет внутрь JSON, а рассылает **исходную строку** `raw` другим участникам комнаты (кроме отправителя). Удобно для экспериментов или особых сценариев, когда логика на клиенте, а сервер только «почтальон».

**Ветки `apply`, `lock`, `chat`, `cursor`, `activity`.** У всех общий шаблон: проверили, что сообщение не **длиннее лимита** (`len(raw) > maxBytes`), проверили, что во внутреннем поле **`d`** что-то есть, и вызвали свою функцию (`dispatchApply`, `dispatchLock`, …). Зачем проверять **длину всего `raw`**, а не только `d`? Потому что злоумышленник мог бы раздуть **другие** поля JSON, обойдя лимит на маленький `d`. Такая проверка — простая страховка.

**Ветка `default` (неизвестный тип).** Если пришло что-то с непонятным `t`, функция возвращает **ошибку**. Тогда `readPump` прекращает чтение и закрывает соединение. Политика жёсткая: **неизвестный протокол — отключаемся**, чтобы не копить мусор и не гадать, что имели в виду.

---

## 5. Apply: конверт `kind` → конкретная операция домена

Когда тип сообщения **`apply`**, внутри `d` лежит уже **конкретная правка сцены**. У этой правки есть своё поле **`kind`** — одно слово вроде `update_widget` или `remove_nls_layer`. Файл **`apply.go`** сначала читает только это одно поле в маленькую структуру **`applyEnvelope`**, а потом большим **`switch`** отправляет работу в отдельную функцию **`apply…Op`**. Сами тяжёлые куски лежат в других файлах (`apply_widget…`, `apply_nls…` и т.д.), чтобы один файл не раздулся на тысячи строк и его было легче читать и тестировать.

```19:97:server/internal/collab/apply.go
const applyOpTimeout = 45 * time.Second

type applyEnvelope struct {
	Kind string `json:"kind"`
}

// applyConnTabIDFromPayload copies clientTabId from the apply JSON into the connection when the WS
// query param was missing (some proxies strip it). Locks and applied fan-out then match the browser tab.
func applyConnTabIDFromPayload(from *Conn, d json.RawMessage) {
	if from == nil || from.clientID != "" {
		return
	}
	var probe struct {
		ClientTabID string `json:"clientTabId"`
	}
	if err := json.Unmarshal(d, &probe); err != nil {
		return
	}
	if x := NormalizeCollabClientID(probe.ClientTabID); x != "" {
		from.clientID = x
	}
}

// ... типы applyUpdateWidget и т.д. ...

func (c *Conn) enqueueJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
	}
}

func dispatchApply(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var head applyEnvelope
	if err := json.Unmarshal(d, &head); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_json", "message": err.Error()}})
		return nil
	}
	applyConnTabIDFromPayload(from, d)

	switch head.Kind {
```

```100:177:server/internal/collab/apply.go
	switch head.Kind {
	case "update_widget":
		return applyUpdateWidgetOp(ctx, hub, from, d)
	case "remove_widget":
		return applyRemoveWidgetOp(ctx, hub, from, d)
	case "add_widget":
		return applyAddWidgetOp(ctx, hub, from, d)
	case "move_story_block":
		return applyMoveStoryBlockOp(ctx, hub, from, d)
	case "create_story_block":
		return applyCreateStoryBlockOp(ctx, hub, from, d)
	case "remove_story_block":
		return applyRemoveStoryBlockOp(ctx, hub, from, d)
	case "create_story_page":
		return applyCreateStoryPageOp(ctx, hub, from, d)
	case "remove_story_page":
		return applyRemoveStoryPageOp(ctx, hub, from, d)
	case "move_story_page":
		return applyMoveStoryPageOp(ctx, hub, from, d)
	case "update_story_page":
		return applyUpdateStoryPageOp(ctx, hub, from, d)
	case "duplicate_story_page":
		return applyDuplicateStoryPageOp(ctx, hub, from, d)
	case "add_nls_layer_simple":
		return applyAddNLSLayerSimpleOp(ctx, hub, from, d)
	case "remove_nls_layer":
		return applyRemoveNLSLayerOp(ctx, hub, from, d)
	case "update_nls_layer":
		return applyUpdateNLSLayerOp(ctx, hub, from, d)
	case "update_nls_layers":
		return applyUpdateNlsLayersOp(ctx, hub, from, d)
	case "create_nls_infobox":
		return applyCreateNLSInfoboxOp(ctx, hub, from, d)
	case "remove_nls_infobox":
		return applyRemoveNLSInfoboxOp(ctx, hub, from, d)
	case "create_nls_photo_overlay":
		return applyCreateNLSPhotoOverlayOp(ctx, hub, from, d)
	case "remove_nls_photo_overlay":
		return applyRemoveNLSPhotoOverlayOp(ctx, hub, from, d)
	case "add_nls_infobox_block":
		return applyAddNLSInfoboxBlockOp(ctx, hub, from, d)
	case "move_nls_infobox_block":
		return applyMoveNLSInfoboxBlockOp(ctx, hub, from, d)
	case "remove_nls_infobox_block":
		return applyRemoveNLSInfoboxBlockOp(ctx, hub, from, d)
	case "update_nls_custom_properties":
		return applyUpdateNlsCustomPropertiesOp(ctx, hub, from, d)
	case "change_nls_custom_property_title":
		return applyChangeNlsCustomPropertyTitleOp(ctx, hub, from, d)
	case "remove_nls_custom_property":
		return applyRemoveNlsCustomPropertyOp(ctx, hub, from, d)
	case "add_nls_geojson_feature":
		return applyAddNLSGeoJSONFeatureOp(ctx, hub, from, d)
	case "update_nls_geojson_feature":
		return applyUpdateNLSGeoJSONFeatureOp(ctx, hub, from, d)
	case "delete_nls_geojson_feature":
		return applyDeleteNLSGeoJSONFeatureOp(ctx, hub, from, d)
	case "add_style":
		return applyAddStyleOp(ctx, hub, from, d)
	case "update_style":
		return applyUpdateStyleOp(ctx, hub, from, d)
	case "remove_style":
		return applyRemoveStyleOp(ctx, hub, from, d)
	case "update_scene_camera":
		return applyUpdateSceneCameraOp(ctx, hub, from, d)
	case "update_property_value":
		return applyUpdatePropertyValueOp(ctx, hub, from, d)
	case "merge_property_json":
		return applyMergePropertyJSONOp(ctx, hub, from, d)
	case "add_property_item":
		return applyAddPropertyItemOp(ctx, hub, from, d)
	case "remove_property_item":
		return applyRemovePropertyItemOp(ctx, hub, from, d)
	case "move_property_item":
		return applyMovePropertyItemOp(ctx, hub, from, d)
	default:
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "unknown_kind", "message": head.Kind}})
		return nil
```

**Подробный разбор (простыми словами):**

**Константа `applyOpTimeout`.** Она объявлена в начале файла как «сколько максимум ждать» при тяжёлых операциях с базой (точное использование — в функциях ниже по файлу). Смысл простой: если Mongo или диск вдруг «зависли», соединение не должно держать горутину **бесконечно** — иначе накопятся зависшие клиенты и упадет память.

**Функция `applyConnTabIDFromPayload`.** Иногда между браузером и сервером стоит **прокси**, который **обрезает** длинный URL и теряет параметр **`clientId`**. Тогда сервер не знает, какая это вкладка. Чтобы не ломать коллаборацию, клиент может положить тот же id во внутрь JSON, в поле вроде **`clientTabId`**. Эта маленькая функция **один раз** подсматривает это поле и, если в соединении ещё не было `clientId`, **копирует** значение в структуру `Conn`. После этого блокировки и сообщения «кто применил правку» снова совпадают с реальной вкладкой.

**Функция `enqueueJSON`.** Ей дают любой объект, она превращает его в JSON и кладёт в очередь на отправку клиенту. Важная деталь — конструкция **`select`** с веткой **`default`**: если канал `send` полон, отправка **не ждёт**, сообщение **выбрасывается**. Это осознанный выбор: **весь хаб** важнее, чем один застрявший клиент. Для ошибок и уведомлений обычно это приемлемо; критичные вещи дублируются другими каналами (например, пользователь может обновить страницу).

**Начало `dispatchApply`.** Сначала пытаются разобрать JSON в **`applyEnvelope`** — там буквально одно поле **`Kind`**. Если JSON битый, соединение **не рвут**: клиенту отправляют сообщение типа **`error`** с кодом `invalid_json` и возвращают **`nil`**. Почему `nil`, а не ошибка? Потому что для `readPump` ошибка означала бы «закрой сокет», а здесь хотят сказать: **«сообщение плохое, но жить можно — жди следующее»**.

**Большой `switch head.Kind`.** Каждая строка `case "…":` — это отдельный **сценарий редактора**: сдвинули блок сторителлинга, удалили слой, обновили стиль и т.д. Внутри соответствующей функции обычно: проверки прав, запись в Mongo, обновление номера версии сцены, рассылка остальным сообщения вроде **`applied`** («вот что изменилось»). Если пришло неизвестное `kind`, клиенту уходит **`error`** с кодом `unknown_kind` — снова без обрыва соединения, чтобы фронт мог показать понятную ошибку.

---

## 6. Клиент: типы входящих сообщений и тонкая обёртка над WebSocket

Сервер шлёт на клиент JSON с полями **`v`** и **`t`**, как и клиент на сервер. На TypeScript удобно описать все допустимые комбинации одним типом **`CollabInbound`**: по сути это список вариантов «если `t` равен такому-то, то в `d` лежат такие-то поля». Редактору проще писать код вида «если пришло `chat` — покажи в панели чата», и редактор подскажет поля, если вы используете IDE.

```3:113:web/src/services/collab/CollabClient.ts
export type CollabInbound =
  | { v: 1; t: "pong" }
  | {
      v: 1;
      t: "presence";
      d?: { event?: string; userId?: string; clientId?: string; photoURL?: string };
    }
  | {
      v: 1;
      t: "presence_snapshot";
      d?: {
        peers?: Array<{
          userId?: string;
          clientId?: string;
          photoURL?: string;
        }>;
      };
    }
  | {
      v: 1;
      t: "lock_changed";
      d?: {
        resource?: string;
        id?: string;
        holderUserId?: string;
        holderClientId?: string;
        until?: string;
        released?: boolean;
      };
    }
  | {
      v: 1;
      t: "lock_denied";
      d?: {
        resource?: string;
        id?: string;
        holderUserId?: string;
        holderClientId?: string;
        until?: string;
      };
    }
  | {
      v: 1;
      t: "chat";
      d?: {
        id?: string;
        userId?: string;
        text?: string;
        ts?: number;
        mentions?: string[];
      };
    }
  | {
      v: 1;
      t: "cursor";
      d?: {
        userId?: string;
        clientId?: string;
        x?: number;
        y?: number;
        inside?: boolean;
        ts?: number;
      };
    }
  | {
      v: 1;
      t: "activity";
      d?: {
        userId?: string;
        clientId?: string;
        kind?: string;
        ts?: number;
      };
    }
  | {
      v: 1;
      t: "applied";
      d?: {
        kind?: string;
        sceneId?: string;
        widgetId?: string;
        userId?: string;
        clientId?: string;
        sceneRev?: number;
        pluginId?: string;
        extensionId?: string;
        opKind?: string;
        layerId?: string;
        layerIds?: string[];
        blockId?: string;
        styleId?: string;
        propertyId?: string;
        fieldId?: string;
        itemId?: string;
        storyId?: string;
        pageId?: string;
      };
    }
  | {
      v: 1;
      t: "notify";
      d?: {
        kind?: string;
        fromUserId?: string;
        messageId?: string;
        text?: string;
        mentions?: string[];
      };
    }
  | { v: 1; t: "error"; d?: { code?: string; message?: string } }
  | { v: 1; t: string; d?: unknown };
```

Класс **`CollabClient`** — это тонкая обёртка над браузерным **`WebSocket`**. Он не знает про React и не знает про виджеты; он умеет: **подключиться** к правильному URL, **отправить** строку JSON, **повесить обработчик** входящих сообщений, **отключиться**. Готовые строки для операций `apply` собираются в другом файле (`applyMessages.ts`), чтобы не дублировать логику и не смешивать «как крутить сокет» с «как описать правку слоя».

```115:168:web/src/services/collab/CollabClient.ts
/** Thin WebSocket helper for /api/collab/ws (ping + optional apply relay). */
export class CollabClient {
  private ws: WebSocket | null = null;

  constructor(
    private readonly apiBase: string,
    private readonly getAccessToken: () => Promise<string>
  ) {}

  get socket(): WebSocket | null {
    return this.ws;
  }

  async connect(projectId: string, clientId?: string): Promise<WebSocket> {
    const token = await this.getAccessToken();
    const url = buildCollabWsUrl(
      this.apiBase,
      projectId,
      token ? token : undefined,
      clientId
    );
    this.ws = new WebSocket(url);
    return this.ws;
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
  }

  ping(): void {
    this.ws?.send(JSON.stringify({ v: 1, t: "ping" }));
  }

  /** Returns false if the socket is not open (caller may queue offline). */
  sendRaw(json: string): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return false;
    }
    this.ws.send(json);
    return true;
  }

  onMessage(handler: (msg: CollabInbound) => void): void {
    if (!this.ws) return;
    this.ws.onmessage = (ev: MessageEvent<string>) => {
      try {
        const data = JSON.parse(ev.data) as CollabInbound;
        handler(data);
      } catch {
        /* ignore malformed */
      }
    };
  }
}
```

**Подробный разбор (простыми словами):**

**Зачем в конструктор передают `getAccessToken`.** Токен доступа может жить в разных местах: в памяти после логина, в библиотеке auth, в тестах — вообще без настоящего пользователя. `CollabClient` не хочет об этом знать: ему дают **функцию «дай строку токена»**, и при **`connect`** он один раз её вызывает. Так один и тот же класс работает и в проде, и в mock-режиме.

**Метод `connect`.** Он **асинхронный**: сначала ждёт токен, потом собирает URL через **`buildCollabWsUrl`**, потом создаёт **`new WebSocket(url)`**. Важно: сам по себе `connect` **не ждёт** успешного handshake в виде Promise — он возвращает объект сокета, который дальше переходит в состояния «соединяется / открыт / ошибка». Остальной код обычно вешает обработчики `onopen` / ошибок в провайдере.

**Метод `sendRaw` и булевый результат.** Он возвращает **`true`**, если строка реально ушла в сокет, и **`false`**, если сокета нет или он ещё не в состоянии **OPEN**. Вызвавший код может тогда **положить операцию в очередь** (`offlineQueue`) и отправить позже, когда сеть восстановится. Это проще, чем кидать исключения на каждый чих.

**Метод `onMessage`.** Вешает стандартный браузерный обработчик **`onmessage`**: внутри делается `JSON.parse`. Если пришла каша (не JSON) — **`try/catch`** молча глотает ошибку. На сервере так обычно не делают: там одна плохая строка может значить атаку. В браузере же мусор иногда прилетает от расширений или прокси, и **рвать весь коллаб из-за одного битого пакета** было бы слишком жестоко для пользователя.

---

## 7. Построение URL WebSocket из базового API

```1:22:web/src/services/collab/collabUrl.ts
/** Build WebSocket URL for real-time collaboration (matches server GET /api/collab/ws). */
export function buildCollabWsUrl(
  apiBase: string,
  projectId: string,
  accessToken?: string,
  /** Per-browser-tab id; server scopes collab locks to this connection. */
  clientId?: string
): string {
  const trimmed = apiBase.replace(/\/$/, "");
  const u = new URL(trimmed);
  u.protocol = u.protocol === "https:" ? "wss:" : "ws:";
  u.pathname = `${u.pathname.replace(/\/$/, "")}/collab/ws`;
  u.searchParams.set("projectId", projectId);
  if (accessToken) {
    u.searchParams.set("access_token", accessToken);
  }
  const cid = clientId?.trim();
  if (cid) {
    u.searchParams.set("clientId", cid);
  }
  return u.toString();
}
```

**Подробный разбор (простыми словами):**

**Откуда берётся адрес.** В `.env` фронта обычно уже есть адрес API, например `http://localhost:8080/api`. Неудобно заводить **второй** адрес специально для WebSocket. Функция берёт этот же адрес, убирает лишний слэш в конце, превращает строку в объект **`URL`**, и дальше делает три простых шага: если было **`http`**, станет **`ws`**; если было **`https`**, станет **`wss`** (шифрование, как у сайта). К пути добавляется **`/collab/ws`** — ровно тот путь, который ждёт сервер в `ServeWS`. Так пользователю не нужно отдельно настраивать «куда стучаться сокетом».

**Зачем токен в строке запроса (`access_token`).** В обычном `fetch` можно повесить заголовок **`Authorization`**. У класса **`WebSocket`** в браузере так сделать **нельзя** так же универсально и просто, как в `fetch`. Поэтому токен кладут в **query** — сервер уже умеет его там читать (как часто делают для сокетов и для скачивания файлов).

**Зачем `clientId` в URL.** Это короткая случайная строка **вкладки**. Она попадает на сервер в `ServeWS` и попадает в `Conn`. Тогда сервер понимает: «это тот же человек, но **другая вкладка**» — и может, например, не давать двум вкладкам одного пользователя держать один и тот же лок, если так задумано продуктом, или наоборот корректно различать их в чате и курсорах.

---

## 8. Ключ «пользователь + вкладка» и подавление уведомлений «сам себе»

```1:60:web/src/services/collab/peerInstanceKey.ts
/** Separates userId and per-tab clientId in composite presence keys. */
export const COLLAB_PEER_SEP = "\u001f";

export function peerInstanceKey(userId: string, clientId?: string): string {
  if (clientId && clientId.length > 0) {
    return `${userId}${COLLAB_PEER_SEP}${clientId}`;
  }
  return userId;
}

export function parsePeerInstanceKey(key: string): {
  userId: string;
  clientId?: string;
} {
  const i = key.indexOf(COLLAB_PEER_SEP);
  if (i < 0) {
    return { userId: key };
  }
  const userId = key.slice(0, i);
  const clientId = key.slice(i + COLLAB_PEER_SEP.length);
  return {
    userId,
    clientId: clientId.length > 0 ? clientId : undefined
  };
}

/** True when peer is this browser tab (same account + same collab client id). */
export function isSameCollabTab(
  localUserId: string | undefined,
  localClientId: string,
  peerUserId: string,
  peerClientId?: string
): boolean {
  if (!localUserId || peerUserId !== localUserId) {
    return false;
  }
  if (!peerClientId) {
    return true;
  }
  return peerClientId === localClientId;
}

/**
 * Skip the "peer applied" toast only when the sender tab matches this tab (same user + same tab key).
 * Uses the same composite key as cursors/presence so it stays consistent with `clientId` on `applied`.
 */
export function suppressCollabPeerAppliedNotification(
  localUserId: string | undefined,
  localClientId: string,
  peerUserId: string,
  peerClientId?: string
): boolean {
  if (!localUserId || peerUserId !== localUserId) {
    return false;
  }
  return (
    peerInstanceKey(peerUserId, peerClientId) ===
    peerInstanceKey(localUserId, localClientId.trim() || undefined)
  );
}
```

**Подробный разбор (простыми словами):**

**Проблема, которую решают эти функции.** Один и тот же человек может открыть **несколько вкладок**. Для программы «Вася» во второй вкладке — это почти «ещё один Вася», но с другим техническим id. Если хранить только `userId`, две вкладки сольются в одну строку в списках и в ключах кэша. Поэтому придумали **составной ключ**: `userId` + редкий символ-разделитель + `clientId`.

**Что за `COLLAB_PEER_SEP`.** Это один невидимый символ с кодом **U+001F** (в коде записан как `\u001f`). Он специально «служебный»: в обычных именах пользователей почти не встречается. Тогда строка вида `user123<разделитель>tab-abc` почти никогда случайно не совпадёт с чужим `userId`.

**`peerInstanceKey` и `parsePeerInstanceKey`.** Первая функция **склеивает** id пользователя и вкладки в одну строку для хранения в словарях. Вторая — **режет** строку обратно: если разделителя нет, значит вкладку не задавали, и ключ — это просто пользователь.

**`isSameCollabTab`.** Отвечает на вопрос: «это сообщение пришло **с этой же вкладки**, что и мы?» Сначала сравнивают пользователей; если у «другого» вообще не указали `clientId`, в коде считают, что это **старый клиент** и ведут себя максимально мягко (считают «та же вкладка», если user совпал).

**`suppressCollabPeerAppliedNotification`.** Это уже про **всплывающие уведомления** («коллега применил изменение»). Если бы мы показывали тост на **каждое** применение, включая **свою же вторую вкладку**, было бы раздражающе. Функция возвращает `true`, когда **не надо** показывать тост: когда пользователь тот же **и** полный ключ `peerInstanceKey` совпал (то есть это **ровно эта вкладка**, а не соседняя). Если у того же пользователя **другая** вкладка — ключи разные — тост **показывают**, потому что с точки зрения UX это действительно «другой сеанс».

---

## 9. Инфраструктура: Mongo в Docker и лимит дескрипторов

Ниже не приложение и не «красивый YAML ради YAML», а кусок **`docker-compose.yml`**, который часто спасает **локальные тесты**. Когда тестов много и они параллельные, Mongo открывает много файлов и сокетов. Если лимит в контейнере маленький, база падает с ошибкой вроде **«слишком много открытых файлов»**, и дальше тесты сыпятся загадочным **connection refused**. Поднятый **`nofile`** — это просто «разреши процессу Mongo держать больше открытых дескрипторов».

```79:97:docker-compose.yml
  reearth-mongo:
    image: mongo:8.0
    hostname: reearth-mongo
    ports:
      - 27017:27017
    volumes:
      # Paths relative to this compose file (repo root), not $PWD — avoids
      # server/tmp/* when compose is run from server/ (breaks `go test ./...`).
      - ./tmp/mongo:/data/db
      - ./mongo-init.js:/docker-entrypoint-initdb.d/mongo-init.js:ro
    command: ["mongod", "--bind_ip_all", "--replSet", "rs0"]
    # Heavy local test suites (e2e + many DBs) can exhaust default nofile and
    # trigger Mongo error 24 "Too many open files", then crashes / connection refused.
    ulimits:
      nofile:
        soft: 65535
        hard: 65535
    networks:
      - reearth
```

**Подробный разбор (простыми словами):**

**Почему путь `./tmp/mongo` именно от корня репозитория.** Docker Compose интерпретирует пути к томам **относительно файла compose**. Если бы том указывал на что-то вроде `server/tmp/...` и при этом вы запускали compose из папки `server/`, путь мог бы «съехать» и данные оказались не там, где их ждут тесты или скрипты. Комментарий в YAML как раз про это: один и тот же compose должен вести себя предсказуемо **неважно, из какой директории его вызвали**.

**Что делает `ulimits.nofile`.** В Linux у процесса есть лимит «сколько одновременно можно держать открытых файлов и сокетов». В контейнере Mongo это касается и журналов, и файлов данных, и клиентских соединений от тестов. Значения **`65535`** — это «подними потолок высоко», чтобы под нагрузкой тестов Mongo **не умерла первой**. Это не «ускоряет тесты», а убирает **случайные** падения.

**Зачем в команде `mongod` указан `--replSet`.** Это настройка режима репликации Mongo. В проекте она может быть нужна фичам или тестам, которые ожидают именно такой режим. Для чтения этого CODE.md достаточно понимать: это **не случайный** флаг, а часть ожидаемого локального стенда.

---

## 10. Как читать этот документ дальше

Здесь в CODE.md показаны **центральные куски**: хаб, соединение, вход по HTTP, разбор типов сообщений, apply, клиентский сокет и URL. Полный список всех операций `apply` — это длинный `switch` в **`server/internal/collab/apply.go`** и много файлов **`apply_*.go`** рядом.

Если интересно, **как React подключает всё это к экрану** (когда переподключаться, что делать при `applied`, как жить офлайн), откройте **`web/src/services/collab/CollabProvider.tsx`**. Файл большой — не читайте его сверху донизу сразу; в редакторе достаточно поиска по словам **`connect`**, **`sendRaw`**, **`applied`**, **`offline`**.

Общая карта проекта — в **[DETAILS.md](DETAILS.md)**, короткий чеклист для нового человека или агента — в **[NEWAGENTS.md](NEWAGENTS.md)**.

**Если номера строк в вставках не совпали с вашей версией кода** — так бывает после любого рефакторинга. Ищите по **имени функции** (`ServeWS`, `dispatchApply`, …) или по **уникальной строке** из кода: это всегда надёжнее, чем полагаться только на цифры в начале блока.
