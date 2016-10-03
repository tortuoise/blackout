package blackout

import (
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/appengine/log"
	"net/http"
	"time"
)

const (
	dateFormat        = "2006-01-02"
	day               = time.Second * 60 * 60 * 24
	defaultCalSummary = "Streaks"
	defaultEvtSummary = "Streak"
	full              = "Full moon"
	last              = "Last quarter"
	newm              = "New moon"
	frst              = "First quarter"
)

var (
	jwtFile = "blackoutmap-54c7be42bcd2.json"
	shade   = map[string]int{full: 255, last: 128, newm: 0, frst: 128}
)

func SAMoonPhases(ctx context.Context) (*[]*calendar.Event, error) {

	saservice, err := GetSACalService(ctx)
	calId, err := calendarId(saservice, "Phases of the Moon", false)
	if err != nil {
		log.Errorf(ctx, "SAMoonPhases: calendarId: %v", err)
		return nil, err
	}

	cal := &Calendar{
		Id:      calId,
		Service: saservice,
	}

	if err = cal.deleteDuplicateEvents(ctx); err != nil {
		log.Errorf(ctx, "SAMoonPhases: deleteDuplicates: %v", err)
		return nil, err
	}
	events := make([]*calendar.Event, 0)
	cal.iterateEvents(ctx, func(e *calendar.Event, start, end time.Time) error {
		events = append(events, e)
		return Continue
	})
	return &events, nil

}

func MoonPhases(c context.Context, client *http.Client) (*[]*calendar.Event, error) {

	service, err := calendar.New(client)
	if err != nil {
		log.Errorf(c, "MoonPhases: service: %v", err)
		return nil, err
	}

	// can copy over events to a default appengine service account calendar so oauth2 user token not required.
	saservice, err := GetSACalService(c)
	if err != nil {
		log.Errorf(c, "MoonPhases: GetSACalService: %v", err)
		return nil, err
	}
	if ok, err := CopyCalendar(c, saservice, service, "Phases of the Moon"); !ok {
		log.Errorf(c, "MoonPhases: CopyCalendar: %v", err)
		return nil, err
	}

	calId, err := calendarId(saservice, "Phases of the Moon", false)
	if err != nil {
		log.Errorf(c, "MoonPhases: calendarId: %v", err)
		return nil, err
	}

	cal := &Calendar{
		Id:      calId,
		Service: saservice,
	}

	events := make([]*calendar.Event, 0)
	cal.iterateEvents(c, func(e *calendar.Event, start, end time.Time) error {
		events = append(events, e)
		return Continue
	})
	return &events, nil

}

//CopyCalendar copies over all the events from one calendar to another - useful to copy over a calendar from a users calendar to the default appengine service account calendar
func CopyCalendar(ctx context.Context, to *calendar.Service, from *calendar.Service, calName string) (bool, error) {

	fromCalId, err := calendarId(from, calName, false)
	if err != nil {
		log.Errorf(ctx, "CopyCalendar from cal: %v", err)
		return false, err
	}
	toCalId, err := calendarId(to, calName, true)
	if err != nil {
		log.Errorf(ctx, "CopyCalendar to cal: %v", err)
		return false, err
	}
	fromCal := &Calendar{
		Id:      fromCalId,
		Service: from,
	}
	toCal := &Calendar{
		Id:      toCalId,
		Service: to,
	}
	n := 0
	if err = fromCal.iterateEvents(ctx, func(e *calendar.Event, start, end time.Time) error {
		_, ierr := toCal.getEvent(e.Id)
		if ierr != nil { // event doesn't already exist
			if start.After(time.Now().AddDate(0, 0, -60)) {
				ierr = toCal.createEvent(e.Summary, start, end)
				if ierr != nil {
					log.Errorf(ctx, "CopyCalendar create event to cal: %v", ierr)
					return ierr //event failed to get added so stop iteration
				}
			}
			n++
		}
		return Continue //event exists so continue iteration
	}); err != nil {
		return false, err
	}
	log.Debugf(ctx, "Copied %v events", n)
	return true, nil

}

func GetSACalService(c context.Context) (*calendar.Service, error) {

	scopes := []string{"https://www.googleapis.com/auth/calendar"}
	client, err := JwtClient(c, jwtFile, scopes...)

	service, err := calendar.New(client)
	if err != nil {
		log.Errorf(c, "GetSACalService: %v", err)
		return nil, err
	}
	return service, nil

	/*calId, err := calendarId(service, "Phases of the Moon", true)
	if err != nil {
		log.Errorf(c, "GetMoonPhases: calendarId: %v", err)
	}

	cal := &Calendar{
		Id:      calId,
		Service: service,
	}

	cal.iterateEvents(c, func(e *calendar.Event, start, end time.Time) error {
		return Continue
	})*/

}

type Calendar struct {
	Id                string
	*calendar.Service //embedded field - files and methods of calendar.Service promoted to Calendar struct
}

var Continue = errors.New("continue")

type iteratorFunc func(e *calendar.Event, start, end time.Time) error

func (c *Calendar) iterateEvents(ctx context.Context, fn iteratorFunc) error {
	var pageToken string
	for {
		call := c.Events.List(c.Id).SingleEvents(true).OrderBy("startTime")
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		events, err := call.Do()
		if err != nil {
			return err
		}
		for _, e := range events.Items {
			if e.Start.Date == "" || e.End.Date == "" { //|| e.Summary != *eventName {
				// Skip non-all-day event or non-streak events.
				//log.Errorf(ctx, "Not all day: %v %v %v", e.Summary, e.Start.Date,e.End.Date)
				continue
			}
			start, end := parseDate(e.Start.Date), parseDate(e.End.Date)
			//log.Errorf(ctx, "All day: %v %v %v", e.Summary, e.Start.Date,e.End.Date)
			if err := fn(e, start, end); err != Continue {
				return err
			}
		}
		pageToken = events.NextPageToken
		if pageToken == "" {
			return nil
		}
	}
}

func (c *Calendar) createEvent(eventName string, start, end time.Time) error {
	e := &calendar.Event{
		//Id: eventId,
		Summary:   eventName,
		Start:     &calendar.EventDateTime{Date: start.Format(dateFormat)},
		End:       &calendar.EventDateTime{Date: end.Format(dateFormat)},
		Reminders: &calendar.EventReminders{UseDefault: false},
	}
	_, err := c.Events.Insert(c.Id, e).Do()
	return err
}

func (c *Calendar) getEvent(eventId string) (*calendar.Event, error) {
	ev, err := c.Events.Get(c.Id, eventId).Do()
	return ev, err
}

func (c *Calendar) deleteDuplicateEvents(ctx context.Context) error {

	var pageToken string
	for {
		call := c.Events.List(c.Id).SingleEvents(true).OrderBy("startTime")
		if pageToken != "" {
			call.PageToken(pageToken)
		}
		events, err := call.Do()
		if err != nil {
			return err
		}
		var prevS, prevE time.Time
		for _, e := range events.Items {
			if e.Start.Date == "" || e.End.Date == "" { //|| e.Summary != *eventName {
				// Skip non-all-day event or non-streak events.
				continue
			}
			start, end := parseDate(e.Start.Date), parseDate(e.End.Date)
			if start == prevS && end == prevE {
				if err := c.Events.Delete(c.Id, e.Id).Do(); err != nil {
					return err
				}
				continue
			}
			prevS, prevE = start, end
		}
		pageToken = events.NextPageToken
		if pageToken == "" {
			return nil
		}
	}

}

func parseDate(s string) time.Time {
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		panic(err)
	}
	return t
}

func calendarId(service *calendar.Service, calName string, create bool) (string, error) {
	list, err := service.CalendarList.List().Do()
	if err != nil {
		return "", err
	}
	for _, entry := range list.Items {
		if entry.Summary == calName {
			return entry.Id, nil
		}
	}

	if create {
		cal, err := service.Calendars.Insert(&calendar.Calendar{Summary: calName}).Do()
		if err != nil {
			return "", err
		}

		return cal.Id, nil
	}

	return "", errors.New(fmt.Sprintf("couldn't find calendar named '%s'", calName))
}

/*func (c *Calendar) addToStreak(today time.Time) (err error) {
	var (
		create = true
		prev   *calendar.Event
	)
	err = c.iterateEvents(func(e *calendar.Event, start, end time.Time) error {
		if prev != nil {
			// We extended the previous event; merge it with this one?
			if prev.End.Date == e.Start.Date {
				// Merge events.
				// Extend this event to begin where the previous one did.
				e.Start = prev.Start
				_, err := c.Events.Update(c.Id, e.Id, e).Do()
				if err != nil {
					return err
				}
				// Delete the previous event.
				return c.Events.Delete(c.Id, prev.Id).Do()
			}
			// We needn't look at any more events.
			return nil
		}
		if start.After(today) {
			if start.Add(-day).Equal(today) {
				// This event starts tomorrow, update it to start today.
				create = false
				e.Start.Date = today.Format(dateFormat)
				_, err = c.Events.Update(c.Id, e.Id, e).Do()
				return err
			}
			// This event is too far in the future.
			return Continue
		}
		if end.After(today) {
			// Today fits inside this event, nothing to do.
			create = false
			return nil
		}
		if end.Equal(today) {
			// This event ends today, update it to end tomorrow.
			create = false
			e.End.Date = today.Add(day).Format(dateFormat)
			_, err := c.Events.Update(c.Id, e.Id, e).Do()
			if err != nil {
				return err
			}
			prev = e
			// Continue to the next event to see if merge is necessary.
		}
		return Continue
	})
	if err == nil && create {
		// No existing events cover or are adjacent to today, so create one.
		err = c.createEvent(today, today.Add(day))
	}
	return
}

func (c *Calendar) removeFromStreak(today time.Time) (err error) {
	err = c.iterateEvents(func(e *calendar.Event, start, end time.Time) error {
		if start.After(today) || end.Before(today) || end.Equal(today) {
			// This event is too far in the future or past.
			return Continue
		}
		if start.Equal(today) {
			if end.Equal(today.Add(day)) {
				// Single day event; remove it.
				return c.Events.Delete(c.Id, e.Id).Do()
			}
			// Starts today; shorten to begin tomorrow.
			e.Start.Date = start.Add(day).Format(dateFormat)
			_, err := c.Events.Update(c.Id, e.Id, e).Do()
			return err
		}
		if end.Equal(today.Add(day)) {
			// Ends tomorrow; shorten to end today.
			e.End.Date = today.Format(dateFormat)
			_, err := c.Events.Update(c.Id, e.Id, e).Do()
			return err
		}

		// Split into two events.
		// Shorten first event to end today.
		e.End.Date = today.Format(dateFormat)
		_, err = c.Events.Update(c.Id, e.Id, e).Do()
		if err != nil {
			return err
		}
		// Create second event that starts tomorrow.
		return c.createEvent(today.Add(day), end)
	})
	return
}*/
