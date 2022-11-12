package main

import (
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/foolin/mixer"
	"github.com/maxence-charriere/go-app/v9/pkg/app"
	"github.com/mitchellh/mapstructure"
	shell "github.com/stateless-minds/go-ipfs-api"
)

const dbAddressEvent = "/orbitdb/bafyreifhynz6quosu65iszr46b6jw3qlfdpgbvqwincnqc72hvhwkan3bm/event"

const (
	topicCreateEvent = "create-event"
	topicUpdateEvent = "update-event"
	eventType        = "event"
)

const (
	NotificationSuccess NotificationStatus = "positive"
	NotificationInfo    NotificationStatus = "info"
	NotificationWarning NotificationStatus = "warning"
	NotificationDanger  NotificationStatus = "negative"
	SuccessHeader                          = "Success"
	ErrorHeader                            = "Error"
)

// pubsub is a component that does a simple pubsub on ipfs. A component is a
// customizable, independent, and reusable UI element. It is created by
// embedding app.Compo into a struct.
type witness struct {
	app.Compo
	sh             *shell.Shell
	sub            *shell.PubSubSubscription
	citizenID      string
	events         []Event
	eventTitle     string
	eventDetails   string
	eventLocation  string
	notifications  map[string]notification
	notificationID int
	noNews         bool
	isWitness      bool
}

type NotificationStatus string

type notification struct {
	id      int
	status  string
	header  string
	message string
}

type Event struct {
	ID          string   `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`
	Type        string   `mapstructure:"type" json:"type" validate:"uuid_rfc4122"`
	ConfirmedBy int      `mapstructure:"confirmedBy" json:"confirmedBy" validate:"uuid_rfc4122"`
	Title       string   `mapstructure:"title" json:"title" validate:"uuid_rfc4122"`
	Details     []string `mapstructure:"details" json:"details" validate:"uuid_rfc4122"`
	Location    string   `mapstructure:"location" json:"location" validate:"uuid_rfc4122"`
	Reporter    string   `mapstructure:"reporter" json:"reporter" validate:"uuid_rfc4122"`
	Witnesses   []string `mapstructure:"witnesses" json:"witnesses" validate:"uuid_rfc4122"`
}

func (w *witness) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	w.sh = sh
	myPeer, err := w.sh.ID()
	if err != nil {
		log.Fatal(err)
	}

	citizenID := myPeer.ID[len(myPeer.ID)-8:]
	// replace password with your own
	password := "mysecretpassword"

	w.citizenID = mixer.EncodeString(password, citizenID)
	w.citizenID = "10"

	w.subscribeToCreateEventTopic(ctx)
	w.subscribeToUpdateEventTopic(ctx)
	w.notifications = make(map[string]notification)

	// set defaults
	w.noNews = true

	ctx.Async(func() {
		// err := w.sh.OrbitDocsDelete(dbAddressEvent, "all")
		// if err != nil {
		// 	log.Fatal(err)
		// }

		v, err := w.sh.OrbitDocsQuery(dbAddressEvent, "type", "event")
		if err != nil {
			log.Fatal(err)
		}

		var vv []interface{}
		err = json.Unmarshal(v, &vv)
		if err != nil {
			log.Fatal(err)
		}

		for _, ii := range vv {
			e := Event{}
			err = mapstructure.Decode(ii, &e)
			if err != nil {
				log.Fatal(err)
			}
			ctx.Dispatch(func(ctx app.Context) {
				if e.ConfirmedBy > 1 {
					w.noNews = false
				}
				for _, v := range e.Witnesses {
					if w.citizenID == v {
						w.isWitness = true
					}
				}
				w.events = append(w.events, e)
				sort.SliceStable(w.events, func(i, j int) bool {
					return w.events[i].ID < w.events[j].ID
				})
			})
		}
	})
}

func (w *witness) subscribeToCreateEventTopic(ctx app.Context) {
	ctx.Async(func() {
		topic := topicCreateEvent
		subscription, err := w.sh.PubSubSubscribe(topic)
		if err != nil {
			log.Fatal(err)
		}
		w.sub = subscription
		w.subscriptionCreateEvent(ctx)
	})
}

func (w *witness) subscribeToUpdateEventTopic(ctx app.Context) {
	ctx.Async(func() {
		topic := topicUpdateEvent
		subscription, err := w.sh.PubSubSubscribe(topic)
		if err != nil {
			log.Fatal(err)
		}
		w.sub = subscription
		w.subscriptionUpdateEvent(ctx)
	})
}

func (w *witness) subscriptionCreateEvent(ctx app.Context) {
	ctx.Async(func() {
		defer w.sub.Cancel()
		// wait on pubsub
		res, err := w.sub.Next()
		if err != nil {
			log.Fatal(err)
		}
		// Decode the string data.
		str := string(res.Data)
		log.Println("Subscriber of topic create-event received message: " + str)
		ctx.Async(func() {
			w.subscribeToCreateEventTopic(ctx)
		})

		e := Event{}
		err = json.Unmarshal([]byte(str), &e)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.events = append(w.events, e)
		})
	})
}

func (w *witness) subscriptionUpdateEvent(ctx app.Context) {
	ctx.Async(func() {
		defer w.sub.Cancel()
		// wait on pubsub
		res, err := w.sub.Next()
		if err != nil {
			log.Fatal(err)
		}
		// Decode the string data.
		str := string(res.Data)
		log.Println("Subscriber of topic update-event received message: " + str)
		ctx.Async(func() {
			w.subscribeToCreateEventTopic(ctx)
		})

		e := Event{}
		err = json.Unmarshal([]byte(str), &e)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			if e.ConfirmedBy > 1 {
				w.noNews = false
			}
			for _, v := range e.Witnesses {
				if w.citizenID == v {
					w.isWitness = true
				}
			}

			id, err := strconv.Atoi(e.ID)
			if err != nil {
				log.Fatal(err)
			}
			w.events[id-1] = e
		})
	})
}

// The Render method is where the component appearance is defined. Here, a
// "pubsub World!" is displayed as a heading.
func (w *witness) Render() app.UI {
	return app.Div().Class("l-application").Role("presentation").Body(
		app.Link().Rel("stylesheet").Href("https://assets.ubuntu.com/v1/vanilla-framework-version-3.8.0.min.css"),
		app.Link().Rel("stylesheet").Href("https://use.fontawesome.com/releases/v6.2.0/css/all.css"),
		app.Link().Rel("stylesheet").Href("/app.css"),
		app.If(len(w.notifications) > 0,
			app.Range(w.notifications).Map(func(s string) app.UI {
				return app.Div().Class("p-notification--"+w.notifications[s].status).Body(
					app.Div().Class("p-notification__content").Body(
						app.H5().Class("p-notification__title").Text(w.notifications[s].header),
						app.P().Class("p-notification__message").Text(w.notifications[s].message),
					),
				).Style("position", "fixed").Style("width", "100%").Style("z-index", "999")
			}),
		),
		app.Section().Class("p-strip--suru").Body(
			app.Div().Class("row u-vertically-center").Body(
				app.Div().Class("col-12").Body(
					app.H1().Text("Cyber Witness - the news as they should be"),
					app.P().Text("P2P community of independent reporters and witnesses - an alternative to mass media. Reporters publish events they have personally seen with no interpretation. Until confirmed they show up as rumors. Witnesses confirm rumors they have witnessed and add their own details. Event details aggregate and become more accurate with the input of each new witness. Once a rumor has been confirmed by at least 2 witnesses it becomes news. The more witnesses the greater accuracy of news."),
					app.Button().Text("How it works").OnClick(w.openHowToDialog),
				),
			),
		),
		app.Section().Class("p-strip--suru").Body(
			app.Div().Class("row u-vertically-center").Body(
				app.Div().Class("col-12").Body(
					app.H1().Text("Have an event to report?"),
					app.Div().Class("p-form p-form--stacked").Body(
						app.Div().Class("p-form__group row").Body(
							// app.Div().Class("p-form__group row").Body(
							app.P().Class("p-form-help-text").ID("reportEvent").Text("Check rumors first as it may already exist.").Style("color", "#fff").Style("margin-top", "0").Style("margin-bottom", "10px"),
							// ),
							app.Label().For("title").Text("Title"),
							app.Input().ID("title").Name("title").OnKeyUp(w.onEventTitle),
						),
						app.Div().Class("p-form__group row").Body(
							app.Label().For("details").Text("Details"),
							app.Textarea().Class("is-dense").ID("details").Name("details").Rows(2).OnKeyUp(w.onEventDetails),
						),
						app.Div().Class("p-form__group row").Body(
							app.Label().For("location").Text("Location"),
							app.Textarea().Class("is-dense").ID("location").Name("location").Rows(2).OnKeyUp(w.onEventLocation),
						),
						// app.Div().Class("p-form__group row").Body(
						// 	app.Label().For("file").Text("Optional Image/Video Evidence"),
						// 	app.Input().Class("is-dense").ID("file").Name("file").Type("file"),
						// ),
						app.Div().Class("p-form__group row").Body(
							app.P().Class("p-form-help-text").ID("reportEvent").Text("Each event that turns out true increases your trust index.").Style("color", "#fff").Style("margin-top", "0"),
						),
						app.Div().Class("p-form__group row").Body(
							app.Button().Class("u-vertically-centered").Text("Report event").OnClick(w.onSubmitEvent),
						),
					),
				),
			),
		).Style("background-image", "linear-gradient(to bottom right, rgba(205, 205, 205, 0.55) 0%, rgba(205, 205, 205, 0.55) 49.8%, transparent 50%, transparent 100%),linear-gradient(to bottom left, rgba(205, 205, 205, 0.55) 0%, rgba(205, 205, 205, 0.55) 49.8%, transparent 50%, transparent 100%),linear-gradient(to top right, #fff 0%, #fff 49%, transparent 50%, transparent 100%),linear-gradient(#fff 0%, #fff 100%),linear-gradient(111deg, #00C6CF 10%, #00C6CF 37%, #00C6CF 100%)"),
		app.Section().Class("p-strip--suru").Body(
			app.Div().Class("row u-vertically-center").Body(
				app.Div().Class("col-12").Body(
					app.H1().Text("Been a witness of an event?"),
					app.P().Text("Confirm a rumor that already exists. Each rumor that turns out true increases your trust index."),
					app.Button().Text("Confirm rumors").OnClick(w.openRumorsDialog),
				),
			),
			app.Div().Class("row u-vertically-center").Body(
				app.Div().Class("col-12").Body(
					app.H1().Text("Just want to read the news?"),
					app.P().Text("Your personal news feed at your fingertips. All witnessed. No ads, paywalls, censorship or fact checkers."),
					app.Button().Text("Read the news").OnClick(w.openNewsDialog),
				),
			),
		).Style("background-image", "linear-gradient(to bottom right, rgba(205, 205, 205, 0.55) 0%, rgba(205, 205, 205, 0.55) 49.8%, transparent 50%, transparent 100%),linear-gradient(to bottom left, rgba(205, 205, 205, 0.55) 0%, rgba(205, 205, 205, 0.55) 49.8%, transparent 50%, transparent 100%),linear-gradient(to top right, #fff 0%, #fff 49%, transparent 50%, transparent 100%),linear-gradient(#fff 0%, #fff 100%),linear-gradient(111deg, #2F4858 10%, #2F4858 37%, #2F4858 100%)"),
		app.Div().Class("p-modal").ID("rumors-modal").Style("display", "none").Body(
			app.Section().Class("p-modal__dialog").Role("dialog").Aria("modal", true).Aria("labelledby", "modal-title").Aria("describedby", "modal-description").Body(
				app.Header().Class("p-modal__header").Body(
					app.H2().Class("p-modal__title").ID("modal-title").Text("Rumors"),
					app.Button().Class("p-modal__close").Aria("label", "Close active modal").Aria("controls", "modal").OnClick(w.closeRumorsModal),
				),
				app.Table().Aria("label", "rumors-table").Class("p-table--expanding").Body(
					app.THead().Body(
						app.Tr().Body(
							app.Th().Body(
								app.Span().Class("status-icon is-blocked").Text("Title"),
							),
							app.Th().Text("Location"),
							app.Th().Text("Action"),
							app.Th().Class("u-align--right").Text("Details"),
						),
					),
					app.If(len(w.events) > 0, app.TBody().Body(
						app.Range(w.events).Slice(func(i int) app.UI {
							return app.Tr().DataSet("title", i).Body(
								app.Td().Class("has-overflow").DataSet("column", "title").Body(
									app.Div().Text(w.events[i].Title),
								),
								app.Td().Class("has-overflow").DataSet("column", "location").Body(
									app.Div().Text(w.events[i].Location),
								),
								app.Td().Class("has-overflow").DataSet("column", "action").Body(
									app.If(w.citizenID == w.events[i].Reporter || w.isWitness,
										app.Button().Class("is-dense").Value(w.events[i].ID).Text("Confirm").Disabled(true).OnClick(w.confirmRumor),
									).Else(
										app.Button().Class("is-dense").Value(w.events[i].ID).Text("Confirm").OnClick(w.confirmRumor),
									),
								),
								app.Td().Class("has-overflow u-align--right").DataSet("column", "details").Body(
									app.Button().Class("u-toggle is-dense").Aria("controls", "expanded-row").Aria("expanded", "true").DataSet("shown-text", "Hide").DataSet("hidden-text", "Show").Value(w.events[i].ID).Text("Hide").OnClick(w.expandDetails),
								),
								app.Td().ID("expanded-row-"+w.events[i].ID).Class("has-overflow p-table__expanding-panel").Aria("hidden", "false").Body(
									app.H4().Text("Details"),
									app.Range(w.events[i].Details).Slice(func(n int) app.UI {
										return app.Div().Class("row").Body(
											app.Div().Class("col-8 p-card").Body(
												app.P().Text(w.events[i].Details[n]),
											),
										)
									}),
									app.If(w.citizenID != w.events[i].Reporter && !w.isWitness,
										app.H4().Text("Add new details: "),
										app.Div().Class("p-form p-form--stacked").Body(
											app.Div().Class("p-form__group row").Body(
												app.Textarea().Class("is-dense").ID("details").Name("details").Rows(2).OnKeyUp(w.onEventDetails),
											),
											app.Div().Class("p-form__group row").Body(
												app.Button().Class("u-vertically-centered").Value(w.events[i].ID).Text("Add details").OnClick(w.onAddDetails),
											),
										),
									),
								),
							)
						}),
					)).Else(
						app.Caption().Class("p-strip").Body(
							app.Div().Class("row").Body(
								app.Div().Class("u-align--left col-8 col-medium-4 col-small-3").Body(
									app.P().Class("p-heading--4 u-no-margin--bottom").Text("No recent rumors"),
									app.P().Text("Check back later or report an event"),
								),
							),
						),
					),
				),
			),
		),
		app.Div().Class("p-modal").ID("news-modal").Style("display", "none").Body(
			app.Section().Class("p-modal__dialog").Role("dialog").Aria("modal", true).Aria("labelledby", "modal-title").Aria("describedby", "modal-description").Body(
				app.Header().Class("p-modal__header").Body(
					app.H2().Class("p-modal__title").ID("modal-title").Text("News"),
					app.Button().Class("p-modal__close").Aria("label", "Close active modal").Aria("controls", "modal").OnClick(w.closeNewsModal),
				),
				app.Table().Aria("label", "news-table").Class("p-table--expanding").Body(
					app.THead().Body(
						app.Tr().Body(
							app.Th().Body(
								app.Span().Class("status-icon is-blocked").Text("Title"),
							),
							app.Th().Text("Location"),
							app.Th().Text("Confirmed By"),
							app.Th().Class("u-align--right").Text("Details"),
						),
					),
					app.If(!w.noNews, app.TBody().Body(
						app.Range(w.events).Slice(func(i int) app.UI {
							return app.If(w.events[i].ConfirmedBy > 1,
								app.Tr().DataSet("title", i).Body(
									app.Td().Class("has-overflow").DataSet("column", "title").Body(
										app.Div().Text(w.events[i].Title),
									),
									app.Td().Class("has-overflow").DataSet("column", "location").Body(
										app.Div().Text(w.events[i].Location),
									),
									app.Td().Class("has-overflow").DataSet("column", "confirmedBy").Body(
										app.Div().Text(w.events[i].ConfirmedBy),
									),
									app.Td().Class("has-overflow u-align--right").DataSet("column", "details").Body(
										app.Button().Class("u-toggle is-dense").Aria("controls", "expanded-row").Aria("expanded", "true").DataSet("shown-text", "Hide").DataSet("hidden-text", "Show").Value(w.events[i].ID).Text("Hide").OnClick(w.expandDetails),
									),
									app.Td().ID("expanded-row-"+w.events[i].ID).Class("has-overflow p-table__expanding-panel").Aria("hidden", "false").Body(
										app.H4().Text("Details"),
										app.Range(w.events[i].Details).Slice(func(n int) app.UI {
											return app.Div().Class("row").Body(
												app.Div().Class("col-8 p-card").Body(
													app.P().Text(w.events[i].Details[n]),
												),
											)
										}),
									),
								),
							)
						}),
					)).Else(
						app.Caption().Class("p-strip").Body(
							app.Div().Class("row").Body(
								app.Div().Class("u-align--left col-8 col-medium-4 col-small-3").Body(
									app.P().Class("p-heading--4 u-no-margin--bottom").Text("No recent news"),
									app.P().Text("Check back later or report an event"),
								),
							),
						),
					),
				),
			),
		),
		app.Div().Class("p-modal").ID("howto-modal").Style("display", "none").Body(
			app.Section().Class("p-modal__dialog").Role("dialog").Aria("modal", true).Aria("labelledby", "modal-title").Aria("describedby", "modal-description").Body(
				app.Header().Class("p-modal__header").Body(
					app.H2().Class("p-modal__title").ID("modal-title").Text("How to play"),
					app.Button().Class("p-modal__close").Aria("label", "Close active modal").Aria("controls", "modal").OnClick(w.closeHowToModal),
				),
				app.Div().Class("p-heading-icon--small").Body(
					app.Aside().Class("p-accordion").Body(
						app.Ul().Class("p-accordion__list").Body(
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab1").Aria("controls", "tab1-section").Aria("expanded", true).Text("What is Cyber Witness").Value("tab1-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab1-section").Aria("hidden", false).Aria("labelledby", "tab1").Body(
									app.P().Text("Cyber Witness is a p2p media simulator based on the reporter and witnesses concept."),
								),
							),
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab2").Aria("controls", "tab2-section").Aria("expanded", true).Text("What's the problem with mass media?").Value("tab2-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab2-section").Aria("hidden", true).Aria("labelledby", "tab1").Body(
									app.P().Text("It's centralized, censored, fact-checked, unaccountable and non-trasparent."),
								),
							),
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab3").Aria("controls", "tab3-section").Aria("expanded", true).Text("How Cyber Witness replaces mass media").Value("tab3-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab3-section").Aria("hidden", true).Aria("labelledby", "tab3").Body(
									app.P().Text("By switching to p2p interactions and the reporter and witnesses model we emulate a transparent environment with a feedback loop."),
								),
							),
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab4").Aria("controls", "tab4-section").Aria("expanded", true).Text("Features").Value("tab4-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab4-section").Aria("hidden", true).Aria("labelledby", "tab3").Body(
									app.Ul().Class("p-matrix").Body(
										app.Li().Class("p-matrix__item").Body(
											app.Div().Class("p-matrix__content").Body(
												app.H3().Class("p-matrix__title").Text("Report events"),
												app.Div().Class("p-matrix__desc").Body(
													app.P().Text("Provide details about the event."),
												),
											),
										),
										app.Li().Class("p-matrix__item").Body(
											app.Div().Class("p-matrix__content").Body(
												app.H3().Class("p-matrix__title").Text("Confirm rumors"),
												app.Div().Class("p-matrix__desc").Body(
													app.P().Text("Browse reported events, confirm what you have witnessed and provide more details."),
												),
											),
										),
										app.Li().Class("p-matrix__item").Body(
											app.Div().Class("p-matrix__content").Body(
												app.H3().Class("p-matrix__title").Text("Event details aggregation"),
												app.Div().Class("p-matrix__desc").Body(
													app.P().Text("The more witnesses the better the accuracy the higher the chance a rumor is real news."),
												),
											),
										),
										app.Li().Class("p-matrix__item").Body(
											app.Div().Class("p-matrix__content").Body(
												app.H3().Class("p-matrix__title").Text("Read the real news"),
												app.Div().Class("p-matrix__desc").Body(
													app.P().Text("Your personal news feed at your fingertips. All witnessed. No ads, paywalls, censorship or fact checkers."),
												),
											),
										),
										app.Li().Class("p-matrix__item").Body(
											app.Div().Class("p-matrix__content").Body(
												app.H3().Class("p-matrix__title").Text("Anonymity by default"),
												app.Div().Class("p-matrix__desc").Body(
													app.P().Text("Anonymity guarantees everyone is protected."),
												),
											),
										),
										app.Li().Class("p-matrix__item").Body(
											app.Div().Class("p-matrix__content").Body(
												app.H3().Class("p-matrix__title").Text("Flat interactions"),
												app.Div().Class("p-matrix__desc").Body(
													app.P().Text("No centralized control, no fact checkers and no ads."),
												),
											),
										),
									),
								),
							),
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab5").Aria("controls", "tab5-section").Aria("expanded", true).Text("Support us").Value("tab5-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab5-section").Aria("hidden", true).Aria("labelledby", "tab5").Body(
									app.A().Href("https://opencollective.com/stateless-minds-collective").Text("https://opencollective.com/stateless-minds-collective"),
								),
							),
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab6").Aria("controls", "tab6-section").Aria("expanded", true).Text("Terms of service").Value("tab6-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab6-section").Aria("hidden", true).Aria("labelledby", "tab6").Body(
									app.Div().Class("p-card").Body(
										app.H3().Text("Introduction"),
										app.P().Class("p-card__content").Text("Cyber Witness is a p2p media simulator based on the reporter and witnesses concept in the form of a fictional game based on real-time data. By using the application you are implicitly agreeing to share your peer id with the IPFS public network."),
									),
									app.Div().Class("p-card").Body(
										app.H3().Text("Application Hosting"),
										app.P().Class("p-card__content").Text("Cyber Witness is a decentralized application and is hosted on a public peer to peer network. By using the application you agree to host it on the public IPFS network free of charge for as long as your usage is."),
									),
									app.Div().Class("p-card").Body(
										app.H3().Text("User-Generated Content"),
										app.P().Class("p-card__content").Text("All published content is user-generated, fictional and creators are not responsible for it."),
									),
								),
							),
							app.Li().Class("p-accordion__group").Body(
								app.Div().Role("heading").Aria("level", "3").Class("p-accordion__heading").Body(
									app.Button().Type("button").Class("p-accordion__tab").ID("tab7").Aria("controls", "tab7-section").Aria("expanded", true).Text("Privacy policy").Value("tab7-section").OnClick(w.toggleAccordion),
								),
								app.Section().Class("p-accordion__panel").ID("tab7-section").Aria("hidden", true).Aria("labelledby", "tab7").Body(
									app.Div().Class("p-card").Body(
										app.H3().Text("Personal data"),
										app.P().Class("p-card__content").Text("There is no personal information collected within Cyber Witness. We store a small portion of your peer ID encrypted as a non-unique identifier which is used for displaying the ranks interface."),
									),
									app.Div().Class("p-card").Body(
										app.H3().Text("Coookies"),
										app.P().Class("p-card__content").Text("Cyber Witness does not use cookies."),
									),
									app.Div().Class("p-card").Body(
										app.H3().Text("Changes to this privacy policy"),
										app.P().Class("p-card__content").Text("This Privacy Policy might be updated from time to time. Thus, it is advised to review this page periodically for any changes. You will be notified of any changes from this page. Changes are effective immediately after they are posted on this page."),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

func (w *witness) onEventTitle(ctx app.Context, e app.Event) {
	w.eventTitle = ctx.JSSrc().Get("value").String()
}

func (w *witness) onEventDetails(ctx app.Context, e app.Event) {
	w.eventDetails = ctx.JSSrc().Get("value").String()
}

func (w *witness) onEventLocation(ctx app.Context, e app.Event) {
	w.eventLocation = ctx.JSSrc().Get("value").String()
}

func (w *witness) onSubmitEvent(ctx app.Context, e app.Event) {
	lastSolutionID := 0
	unique := true
	for n, ev := range w.events {
		if w.eventTitle == ev.Title {
			unique = false
		}

		if n > 0 {
			currentID, err := strconv.Atoi(ev.ID)
			if err != nil {
				log.Fatal(err)
			}
			previousID, err := strconv.Atoi(w.events[n-1].ID)
			if err != nil {
				log.Fatal(err)
			}
			if currentID > previousID {
				lastSolutionID = currentID
			}
		} else {
			lastSolutionID = 1
		}
	}

	if unique {
		event := Event{
			ID:       strconv.Itoa(lastSolutionID + 1),
			Type:     eventType,
			Title:    w.eventTitle,
			Location: w.eventLocation,
			Reporter: w.citizenID,
		}

		event.Details = append(event.Details, w.eventDetails)

		ev, err := json.Marshal(event)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Async(func() {
			err = w.sh.OrbitDocsPut(dbAddressEvent, ev)
			if err != nil {
				ctx.Dispatch(func(ctx app.Context) {
					w.createNotification(ctx, NotificationDanger, ErrorHeader, "Could not create event. Try again later.")
					log.Fatal(err)
				})
			}
			err = w.sh.PubSubPublish(topicCreateEvent, string(ev))
			if err != nil {
				log.Fatal(err)
			}

			ctx.Dispatch(func(ctx app.Context) {
				w.createNotification(ctx, NotificationSuccess, SuccessHeader, "Event submited.")
			})
		})
	}
}

func (w *witness) onAddDetails(ctx app.Context, e app.Event) {
	id := ctx.JSSrc().Get("value").String()
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Fatal(err)
	}

	// add new details to slice
	w.events[idInt-1].Details = append(w.events[idInt-1].Details, w.eventDetails)

	ev, err := json.Marshal(w.events[idInt-1])
	if err != nil {
		log.Fatal(err)
	}

	ctx.Async(func() {
		err = w.sh.OrbitDocsPut(dbAddressEvent, ev)
		if err != nil {
			ctx.Dispatch(func(ctx app.Context) {
				w.createNotification(ctx, NotificationDanger, ErrorHeader, "Could not add details. Try again later.")
				log.Fatal(err)
			})
		}
		err = w.sh.PubSubPublish(topicUpdateEvent, string(ev))
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.createNotification(ctx, NotificationSuccess, SuccessHeader, "Event details added.")
		})
	})
}

func (w *witness) createNotification(ctx app.Context, s NotificationStatus, h string, msg string) {
	w.notificationID++
	w.notifications[strconv.Itoa(w.notificationID)] = notification{
		id:      w.notificationID,
		status:  string(s),
		header:  h,
		message: msg,
	}

	ntfs := w.notifications
	ctx.Async(func() {
		for n := range ntfs {
			time.Sleep(5 * time.Second)
			delete(ntfs, n)
			ctx.Async(func() {
				ctx.Dispatch(func(ctx app.Context) {
					w.notifications = ntfs
				})
			})
		}
	})
}

func (w *witness) openRumorsDialog(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("rumors-modal").Set("style", "display:flex")
}

func (w *witness) openNewsDialog(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("news-modal").Set("style", "display:flex")
}

func (w *witness) openHowToDialog(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("howto-modal").Set("style", "display:flex")
}

func (w *witness) closeRumorsModal(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("rumors-modal").Set("style", "display:none")
}

func (w *witness) closeNewsModal(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("news-modal").Set("style", "display:none")
}

func (w *witness) closeHowToModal(ctx app.Context, e app.Event) {
	app.Window().GetElementByID("howto-modal").Set("style", "display:none")
}

func (w *witness) expandDetails(ctx app.Context, e app.Event) {
	id := ctx.JSSrc().Get("value").String()
	attrButton := ctx.JSSrc().Get("attributes")
	dataShownText := attrButton.Get("data-shown-text").Get("value").String()
	dataHiddenText := attrButton.Get("data-hidden-text").Get("value").String()
	ariaButton := attrButton.Get("aria-expanded").Get("value").String()
	attrRow := app.Window().GetElementByID("expanded-row-" + id).Get("attributes")
	ariaRow := attrRow.Get("aria-hidden").Get("value").String()

	if ariaButton == "false" {
		ctx.JSSrc().Call("setAttribute", "aria-expanded", "true")
		ctx.JSSrc().Set("innerHTML", dataShownText)
	} else {
		ctx.JSSrc().Call("setAttribute", "aria-expanded", "false")
		ctx.JSSrc().Set("innerHTML", dataHiddenText)
	}

	if ariaRow == "false" {
		app.Window().GetElementByID("expanded-row-"+id).Call("setAttribute", "aria-hidden", "true")
	} else {
		app.Window().GetElementByID("expanded-row-"+id).Call("setAttribute", "aria-hidden", "false")
	}
}

func (w *witness) confirmRumor(ctx app.Context, e app.Event) {
	id := ctx.JSSrc().Get("value").String()
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Fatal(err)
	}

	event := w.events[idInt-1]

	if w.citizenID != event.Reporter {
		// increment confirmedBy counter
		event.ConfirmedBy++
		event.Witnesses = append(event.Witnesses, w.citizenID)
		w.isWitness = true
	} else {
		// return if reporter somehow made a request
		return
	}

	ev, err := json.Marshal(event)
	if err != nil {
		log.Fatal(err)
	}

	ctx.Async(func() {
		err = w.sh.OrbitDocsPut(dbAddressEvent, ev)
		if err != nil {
			ctx.Dispatch(func(ctx app.Context) {
				w.createNotification(ctx, NotificationDanger, ErrorHeader, "Could not confirm rumor. Try again later.")
				log.Fatal(err)
			})
		}
		err = w.sh.PubSubPublish(topicUpdateEvent, string(ev))
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			w.events[idInt-1] = event
			w.createNotification(ctx, NotificationSuccess, SuccessHeader, "Rumor confirmed.")
		})
	})
}

func (w *witness) toggleAccordion(ctx app.Context, e app.Event) {
	id := ctx.JSSrc().Get("value").String()
	attr := app.Window().GetElementByID(id).Get("attributes")
	aria := attr.Get("aria-hidden").Get("value").String()
	if aria == "false" {
		app.Window().GetElementByID(id).Call("setAttribute", "aria-hidden", "true")
	} else {
		app.Window().GetElementByID(id).Call("setAttribute", "aria-hidden", "false")
	}
}
