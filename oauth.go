package blackout

import (
        "fmt"
	"io/ioutil"
        "encoding/json"
	"net/http"
        "time"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/net/context"
        "cloud.google.com/go/storage"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/file"
)

const (

)

var bucket = "blackoutmap.appspot.com"
var randState string
var ch = make(chan string)
var cfg *oauth2.Config

func JwtClient(c context.Context, jsonFilename string, scopes ...string) (*http.Client, error ) {

        jwtJSONKey, err := ioutil.ReadFile(jsonFilename)
        if err != nil {
                log.Errorf(c, "Error in reading from json file: %v", err)
                return nil, err
        }
        conf, err := google.JWTConfigFromJSON(jwtJSONKey, scopes...)
        if err != nil {
                log.Errorf(c, "Error in config from json file: %v", err)
                return nil, err
        }
        //client := conf.Client(oauth2.NoContext)
        client := conf.Client(c)
        return client, nil

}

func oauth2Config(jsonFilename string, scopes ...string) (*oauth2.Config, error ) {

        oauth2JSONKey, err := ioutil.ReadFile(jsonFilename)
        if err != nil {
                log.Errorf(context.Background(), "Error in reading from json file: %v", err)
                return nil, err
        }
        conf, err := google.ConfigFromJSON(oauth2JSONKey, scopes...)
        if err != nil {
                log.Errorf(context.Background(), "Error in config from json: %v", err)
                return nil, err
        }
        return conf, nil

}

func OauthClientToken(c context.Context, cfg *oauth2.Config, code string) (*http.Client, *oauth2.Token) {

        log.Errorf(c, "Exchanging code for token: %v", code)
        tok, err := cfg.Exchange(c, code)
        if err != nil {
                log.Errorf(c, "Error in exchange: %v", err)
                return nil, nil
        }
        client := cfg.Client(c, tok)
        return client, tok
        /*log.Errorf(c, " Selecting ... : %v", "")
        select {
        case code := <-ch:
                log.Errorf(c, "Exchanging code for token: %v", code)
                tok, err := cfg.Exchange(oauth2.NoContext, code)
                if err != nil {
                        log.Errorf(c, "Error in exchange: %v", err)
                }
                client := cfg.Client(oauth2.NoContext, tok)
                return client, tok

        case <- time.After(60000 * time.Millisecond):
                log.Errorf(c, " Timing out after 5 seconds: %v", "Stroof")
                return nil, nil

        }*/
}

func AuthCodeURL(jsonFilename string, scopes ...string) (*oauth2.Config, string) {

	randState = fmt.Sprintf("st%d", time.Now().UnixNano())

	cfg, err := oauth2Config(jsonFilename, scopes...)
	if err != nil {
		return nil, ""
	}
	url := cfg.AuthCodeURL(randState, oauth2.AccessTypeOffline)

        return cfg, url

}

//WriteCloudToken writes the token provided as argument to cloud storage with name provided as argument
func WriteCloudToken(ctx context.Context, mth *oauth2.Token, filename string) error {

        var err error
	//[START get_default_bucket]
	if bucket == "" {
		if bucket, err = file.DefaultBucketName(ctx); err != nil {
			log.Errorf( ctx, "failed to get default GCS bucket name: %v\n", err)
			return err
		}
	}
	//[END get_default_bucket]
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf(ctx, "failed to create client: %v\n", err)
		return err
	}
        defer client.Close()
	wc := client.Bucket(bucket).Object(filename).NewWriter(ctx)
	wc.ContentType = "text/json"
	wc.ACL = []storage.ACLRule{{storage.AllUsers, storage.RoleReader}}
        enc := json.NewEncoder(wc)
        if err = enc.Encode(mth); err != nil {
		log.Errorf(ctx,  "failed to write: %v\n", err)
		return err
	}
        defer wc.Close()
	log.Errorf(ctx, "updated object: %v\n", wc.Attrs())

        return err

}

//ReadCloudToken reads the token with filename as argument stored in GCS bucket
func ReadCloudToken(ctx context.Context, filename string) (*oauth2.Token, error) {

        var err error
	//[START get_default_bucket]
	if bucket == "" {
		if bucket, err = file.DefaultBucketName(ctx); err != nil {
			log.Errorf(ctx, "failed to get default GCS bucket name: %v\n", err)
			return nil, err
		}
	}
	//[END get_default_bucket]
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Errorf(ctx, "failed to create client: %v\n", err)
		return nil, err
	}
        defer client.Close()

	rc, err := client.Bucket(bucket).Object(filename).NewReader(ctx)
	if err != nil {
		log.Errorf(ctx, "readFile: unable to open file from bucket %q, file %q: %v", bucket, filename, err)
		return nil, err
	}
	defer rc.Close()

        dec := json.NewDecoder(rc)

        var tok oauth2.Token
	err = dec.Decode(&tok)
	if err != nil {
		log.Errorf(ctx, "readFile: unable to read data from bucket %q, file %q: %v", bucket, filename, err)
		return nil, err
	}

        return &tok, nil
}
