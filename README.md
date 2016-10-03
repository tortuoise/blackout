<A name="toc0_1" title="Blackout"/>
# Client/Server 

##Contents     
**<a href="toc1_1">Google Service Account using JWT Authentication</a>**  
**<a href="toc1_2">Google oAuth2 Client Authentication</a>**  
**<a href="toc1_3">Context</a>**  
**<a href="toc1_4">Calendar API</a>**   
**<a href="toc1_5">Go image pkg</a>**   
**<a href="toc1_6">References</a>**   

<A name="toc1_1" title="Service Account using JWT " />
## Service Account using JWT ##
The default appengine service account is used by apps to access Google APIs when no user data is required. This is suitable for the "Phases of the Moon" (moon) calendar as no user calendars are required. The JWT is stored in a file and is used as the primary access method of moon calendar events. 
[Google JWT](https://godoc.org/golang.org/x/oauth2/google)

<A name="toc1_2" title="Client oAuth2" />
## 3 legged Client oAuth2 ##
While service account JWT is useful for instances when no user calendars are required, the service account has no calendars to begin with and doesn't have access to the common calendars available to a user. So the moon calendar must be copied over once from a user account to the service account. If the service account moon calendar doesn't exist or doesn't have any events, the user is prompted to allow offline access to the "Phases of the Moon" calendar in the user account using 3 legged oAuth2. The token received is then written to the default GCS bucket from where it can be retrieved whenever new events need to be added to the service account moon calendar.
[Google oAuth2](https://godoc.org/golang.org/x/oauth2)

<A name="toc1_3" title="Context" />
## Context ##

[Appengine context](https://godoc.org/appengine/context)
[Go context](https://godoc.org/golang.org/x/net/context)

<A name="toc1_4" title="Calendar API" />
## The Calendar API ##
The calendar is accessed via a service which encapsulates an authenticated *http.Client (either JWT or oAuth2).
[Google Calendar API](https://godoc.org/google.golang.org/api/calendar/v3)

<A name="toc1_5" title="Go image pkg" />
## Go image pkg ##
The image package 

Testing with the dev_appserver needs environment variable $APPENGINE_DEV_APPSERVER set. Run tests as usual with go test ./server
<A name="toc1_5" title="References" />
## References ##
[Making a restful JSON API in Go](http://thenewstack.io/make-a-restful-json-api-go/)
[Appengine logging](https://cloud.google.com/appengine/docs/go/logs/)
[Cloud Storage API](https://godoc.org/cloud.google.com/go/storage)

