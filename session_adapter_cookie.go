package revel

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/revel/revel/session"
)

type (
	Session = session.Session
	
	// The session cookie engine
	SessionCookieEngine struct {
		ExpireAfterDuration time.Duration
	}
)

var (
	// A logger for the session engine
	sessionEngineLog = RevelLog.New("section", "session-engine")

	cookiePath string
	sessionKey = CookiePrefix + session.SessionCookieSuffix
)

// Create a new instance to test
func init() {
	RegisterSessionEngine(initCookieEngine, "revel-cookie")
}

func SetCookiePath(pa string) {
	cookiePath = pa
}
func SetSessionKey(key string) {
	sessionKey = key
}

// For testing purposes this engine is used
func NewSessionCookieEngine() *SessionCookieEngine {
	ce := &SessionCookieEngine{}
	return ce
}

// Called when the the application starts, retrieves data from the app config so cannot be run until then
func initCookieEngine() SessionEngine {
	ce := &SessionCookieEngine{}

	var err error
	if expiresString, ok := Config.String("session.expires"); !ok {
		ce.ExpireAfterDuration = 30 * 24 * time.Hour
	} else if expiresString == session.SessionValueName {
		ce.ExpireAfterDuration = 0
	} else if ce.ExpireAfterDuration, err = time.ParseDuration(expiresString); err != nil {
		panic(fmt.Errorf("session.expires invalid: %s", err))
	}

	return ce
}

// Decode the session information from the cookie retrieved from the controller request
func (cse *SessionCookieEngine) Decode(c *Controller) {
	// Decode the session from a cookie.
	c.Session = map[string]interface{}{}
	sessionMap := c.Session
	if cookie, err := c.Request.Cookie(sessionKey); err != nil {
		return
	} else {
		cse.DecodeCookie(cookie, sessionMap)
		c.Session = sessionMap
	}
}

// Encode the session information to the cookie, set the cookie on the controller
func (cse *SessionCookieEngine) Encode(c *Controller) {

	c.SetCookie(cse.GetCookie(c.Session))
}

// Exposed only for testing purposes
func (cse *SessionCookieEngine) DecodeCookie(cookie ServerCookie, s session.Session) {
	// Decode the session from a cookie.
	// Separate the data from the signature.
	cookieValue := cookie.GetValue()
	hyphen := strings.Index(cookieValue, "-")
	if hyphen == -1 || hyphen >= len(cookieValue)-1 {
		return
	}
	sig, data := cookieValue[:hyphen], cookieValue[hyphen+1:]

	// Verify the signature.
	if !Verify(data, sig) {
		sessionEngineLog.Warn("Session cookie signature failed")
		return
	}

	// Parse the cookie into a temp map, and then load it into the session object
	tempMap := map[string]string{}
	ParseKeyValueCookie(data, func(key, val string) {
		tempMap[key] = val
	})
	s.Load(tempMap)

	// Check timeout after unpacking values - if timeout missing (or removed) destroy all session
	// objects
	if s.SessionTimeoutExpiredOrMissing() {
		// If this fails we need to delete all the keys from the session
		for key := range s {
			delete(s, key)
		}
	}
}

// Convert session to cookie
func (cse *SessionCookieEngine) GetCookie(s session.Session) *http.Cookie {
	var sessionValue string
	ts := s.GetExpiration(cse.ExpireAfterDuration)
	if ts.IsZero() {
		s[session.TimestampKey] = session.SessionValueName
	} else {
		s[session.TimestampKey] = strconv.FormatInt(ts.Unix(), 10)
	}

	// Convert the key to a string map
	stringMap := s.Serialize()

	for key, value := range stringMap {
		if strings.ContainsAny(key, ":\x00") {
			panic("Session keys may not have colons or null bytes")
		}
		if strings.Contains(value, "\x00") {
			panic("Session values may not have null bytes")
		}
		sessionValue += "\x00" + key + ":" + value + "\x00"
	}

	if len(sessionValue) > 1024*4 {
		sessionEngineLog.Error("SessionCookieEngine.Cookie, session data has exceeded 4k limit (%d) cookie data will not be reliable", "length", len(sessionValue))
	}

	sessionData := url.QueryEscape(sessionValue)
	sessionCookie := &http.Cookie{
		Name:     sessionKey,
		Value:    Sign(sessionData) + "-" + sessionData,
		Domain:   CookieDomain,
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   CookieSecure,
		SameSite: CookieSameSite,
		Expires:  ts.UTC(),
		MaxAge:   int(cse.ExpireAfterDuration.Seconds()),
	}

	return sessionCookie
}
