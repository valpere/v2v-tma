package domain

type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceUAH    int    `json:"price_uah"`
	PriceStars  int    `json:"price_stars"`
	Category    string `json:"category"`
	Icon        string `json:"icon"`
	Badge       string `json:"badge,omitempty"`
}

type DomainInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tagline     string   `json:"tagline"`
	Icon        string   `json:"icon"`
	Welcome     string   `json:"welcome"`
	VoiceSample string   `json:"voice_sample"`
	Prompts     []string `json:"quick_prompts"`
	Items       []Item   `json:"items"`
}

type UIAction struct {
	Type    string      `json:"type"` // HIGHLIGHT_ITEM, ADD_TO_CART, REMOVE_FROM_CART, NOTIFY, OPEN_CHECKOUT
	Payload interface{} `json:"payload"`
}

type ChatResponse struct {
	ReplyText string     `json:"reply_text"`
	Language  string     `json:"language"`
	Intent    string     `json:"intent"`
	DomainID  string     `json:"domain_id"`
	AudioURL  string     `json:"audio_url,omitempty"`
	UIActions []UIAction `json:"ui_actions"`
}

type Repository struct {
	domains map[string]DomainInfo
}

func NewRepository() *Repository {
	r := &Repository{
		domains: make(map[string]DomainInfo),
	}
	r.initCatalog()
	return r
}

func (r *Repository) GetDomain(id string) (DomainInfo, bool) {
	d, ok := r.domains[id]
	return d, ok
}

func (r *Repository) ListDomains() []DomainInfo {
	list := make([]DomainInfo, 0, len(r.domains))
	for _, d := range r.domains {
		list = append(list, d)
	}
	return list
}

func (r *Repository) initCatalog() {
	// 1. Pizzeria
	r.domains["pizzeria"] = DomainInfo{
		ID:          "pizzeria",
		Name:        "Піцерія «Папа Пепе»",
		Tagline:     "Справжня неаполітанська піца на дровах",
		Icon:        "🍕",
		Welcome:     "Вітаю в «Папа Пепе»! Що для вас приготувати? Можу порадити піцу з хрустким бортиком або зібрати сет для компанії.",
		VoiceSample: "Привіт! Я голосовий консьєрж Папа Пепе. Чим можу допомогти?",
		Prompts: []string{
			"Порадь гостру піцу",
			"Додай Пепероні та Колу",
			"Скільки коштує Маргарита?",
			"Оформити доставку",
		},
		Items: []Item{
			{
				ID:          "pizza_peperoni",
				Title:       "Піца Пепероні",
				Description: "Пікантна салямі, моцарела fior di latte, томатний соус San Marzano, базилік",
				PriceUAH:    340,
				PriceStars:  35,
				Category:    "Піца",
				Icon:        "🍕",
				Badge:       "Хіт продажу",
			},
			{
				ID:          "pizza_margherita",
				Title:       "Піца Маргарита",
				Description: "Класичний томатний соус, подвійна моцарела, оливкова олія першого віджиму",
				PriceUAH:    280,
				PriceStars:  30,
				Category:    "Піца",
				Icon:        "🧀",
			},
			{
				ID:          "pizza_quattro_formaggi",
				Title:       "Чотири Сири",
				Description: "Моцарела, горгонзола, пармезан, рікота, трюфельний мед",
				PriceUAH:    390,
				PriceStars:  40,
				Category:    "Піца",
				Icon:        "🧀",
				Badge:       "Преміум",
			},
			{
				ID:          "drink_cola",
				Title:       "Coca-Cola 0.5л",
				Description: "Охолоджений класичний напій у склі",
				PriceUAH:    60,
				PriceStars:  6,
				Category:    "Напої",
				Icon:        "🥤",
			},
			{
				ID:          "sauce_garlic",
				Title:       "Часниковий соус айолі",
				Description: "Фірмовий домашній соус до бортиків",
				PriceUAH:    45,
				PriceStars:  5,
				Category:    "Соуси",
				Icon:        "🧄",
			},
		},
	}

	// 2. Barbershop
	r.domains["barbershop"] = DomainInfo{
		ID:          "barbershop",
		Name:        "Барбершоп «Барбароса»",
		Tagline:     "Чоловічі стрижки та королівське гоління",
		Icon:        "💈",
		Welcome:     "Вітаю в «Барбароса»! Допоможу обрати послугу, майстра або вільний слот на сьогодні.",
		VoiceSample: "Барбершоп Барбароса слухає. На яку послугу бажаєте записатися?",
		Prompts: []string{
			"Запиши на стрижку бороди",
			"Скільки коштує комплекс?",
			"Вільний час на сьогодні",
		},
		Items: []Item{
			{
				ID:          "haircut_classic",
				Title:       "Чоловіча стрижка",
				Description: "Мирове миття голови, індивідуальний підбір стилю, укладання преміум-стайлінгом",
				PriceUAH:    550,
				PriceStars:  55,
				Category:    "Стрижка",
				Icon:        "✂️",
				Badge:       "Популярне",
			},
			{
				ID:          "beard_trim",
				Title:       "Моделювання бороди",
				Description: "Розпарювання рушником, чіткі контури небезпечною бритвою, бальзам з ароматом кедра",
				PriceUAH:    400,
				PriceStars:  40,
				Category:    "Борода",
				Icon:        "🧔",
			},
			{
				ID:          "combo_royal",
				Title:       "Комплекс Стрижка + Борода",
				Description: "Повний догляд: стрижка волосся, моделювання бороди та камуфляж сивини за бажанням",
				PriceUAH:    850,
				PriceStars:  85,
				Category:    "Комплекс",
				Icon:        "👑",
				Badge:       "-10%",
			},
		},
	}

	// 3. Auto Service
	r.domains["auto"] = DomainInfo{
		ID:          "auto",
		Name:        "Автосервіс «Подбай-Авто»",
		Tagline:     "Комп'ютерна діагностика та сервісне ТО",
		Icon:        "🚗",
		Welcome:     "Вітаю в «Подбай-Авто»! Яка проблема з авто або яке планове обслуговування потрібно провести?",
		VoiceSample: "Автосервіс Подбай-Авто. Що саме турбує у вашому автомобілі?",
		Prompts: []string{
			"Компʼютерна діагностика",
			"Скільки коштує заміна мастила?",
			"Перевірити гальмівну систему",
		},
		Items: []Item{
			{
				ID:          "auto_diag_pc",
				Title:       "Комп'ютерна діагностика",
				Description: "Повне сканування блоків керування ECU, скидання помилок та аналіз параметрів датчиків",
				PriceUAH:    500,
				PriceStars:  50,
				Category:    "Діагностика",
				Icon:        "💻",
			},
			{
				ID:          "auto_oil_service",
				Title:       "Заміна мастила та фільтрів",
				Description: "Експрес-заміна моторного мастила, масляного, повітряного та салонного фільтрів",
				PriceUAH:    600,
				PriceStars:  60,
				Category:    "ТО",
				Icon:        "🛢️",
			},
			{
				ID:          "auto_brake_check",
				Title:       "Діагностика ходової та гальм",
				Description: "Перевірка товщини колодок, стану дисків, супортів та елементів підвіски на підйомнику",
				PriceUAH:    450,
				PriceStars:  45,
				Category:    "Ходова",
				Icon:        "⚙️",
			},
		},
	}

	// 4. Lyapko Store
	r.domains["lyapko"] = DomainInfo{
		ID:          "lyapko",
		Name:        "Аплікатори «Lyapko Shop»",
		Tagline:     "Оригінальні голчасті масажери та валики лікаря Ляпка",
		Icon:        "🛍️",
		Welcome:     "Вітаю в магазині продукції Ляпка! Допоможу підібрати аплікатор для спини, шиї чи зняття втоми.",
		VoiceSample: "Вітаю! Підкажу, який крок голки та розмір аплікатора підійде саме вам.",
		Prompts: []string{
			"Який аплікатор для спини?",
			"Чим відрізняються валики?",
			"Додай Валик Великий",
		},
		Items: []Item{
			{
				ID:          "lyapko_mat_large",
				Title:       "Килимок «Великий» 7.0 Ag",
				Description: "Універсальний килимок для спини (275х480 мм). Голки: мідь, залізо, цинк, нікель, срібло",
				PriceUAH:    1850,
				PriceStars:  185,
				Category:    "Килимки",
				Icon:        "🛏️",
				Badge:       "Топ для спини",
			},
			{
				ID:          "lyapko_roller_big",
				Title:       "Валик «Великий» 5.0 Ag",
				Description: "Динамічний масаж для великих ділянок тіла: спина, поперек, сідниці, стегна",
				PriceUAH:    890,
				PriceStars:  90,
				Category:    "Валики",
				Icon:        "🪄",
			},
		},
	}
}
