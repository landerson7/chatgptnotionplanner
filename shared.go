package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	httpswag "github.com/swaggo/http-swagger"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
	"github.com/spf13/viper"
	"github.com/theothertomelliott/acyclic"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

///////////////////////////////////////////////////////////////////////////////////
//////// Global Shared Standalone API Variable Definitions
///////////////////////////////////////////////////////////////////////////////////

var defaultApiTimezone string = "UTC"
var thisExecTestingOnly bool = false
var thisApiKey string = ""
var thisApiServicePrefix string = ""
var thisApiServicePrefixLen int = 0
var thisApiTimezone string = defaultApiTimezone
var thisAppVersion string = "0.0.0"

var thisAcnsApiKey string = ""
var thisAcnsApiDomain string = ""
var thisAcnsClientName string = ""
var thisAuthedBearerTokens map[string]bool = make(map[string]bool)
var thisCheckPhoneNumberCountries []string = []string{
	"CA", // Canada
	"US",
}
var thisCpaasEmulatorApiKey string = ""
var thisCpaasEmulatorDomain string = ""
var thisGcsBaseSavePath string = ""
var thisGcsStorageBucket string = ""
var thisGoogleTokenAuthEmails []string = make([]string, 0)
var thisHostname string = ""
var thisPageFavicon string = ""
var thisPubSubProjectId string = ""
var thisPubSubTopicId string = ""
var thisRedisDbConnected bool = false
var thisRedisDbEnabled bool = false
var thisRedisDbTlsEnabled bool = false
var thisRedisDbError error = nil
var thisRedisApiKey string = ""
var thisRedisApiUrl string = ""
var thisTtsAllVoicesList MapTtsVoicesList
var thisTtsApiKey string = ""
var thisTtsApiUrl string = ""
var thisTtsAudioSavePath string = ""
var thisTtsDefaultVoices map[string]interface{} = make(map[string]interface{})
var thisTtsStorageBucket string = ""
var indexRegex string = "(/(index\\.(html|php|asp|js|cgi|xhtml|htm|pl))?)?"

var redisHost string = ""
var redisPort string = "6379"
var redisPass string = ""
var rdb *redis.Client
var rdbCtx context.Context
var rdbCtxExit context.CancelFunc

var logger *zap.Logger
var KnownCommonBrowsers []string = []string{
	"Mozilla",
	"Safari",
	"Firefox",
	"Chrome",
	"Gecko",
	"OPR",
	"Vivaldi",
	"Trident",
	"Edg",
	"Edge",
	"MSIE",
	"Yowser",
	"YaBrowser",
	"Macintosh",
	"Windows",
	"Apple",
	"iOS",
}

///////////////////////////////////////////////////////////////////////////////////
//////// Global Shared Standalone API Struct Definitions
///////////////////////////////////////////////////////////////////////////////////

type TraceType struct {
	Frame int
	File  string
	Line  int
	Func  string
} // @name TraceType

type TraceSlice struct {
	Stack []TraceType
} // @name TraceSlice

type route struct {
	pattern *regexp.Regexp
	handler http.Handler
} // @name route

type RegexpHandler struct {
	routes []*route
} // @name RegexpHandler

type RequestUri struct {
	Method  string `json:"method,omitempty"`
	BaseUrl string `json:"base_url,omitempty"`
	Scheme  string `json:"scheme,omitempty"`
	Domain  string `json:"domain,omitempty"`
	Port    string `json:"port,omitempty"`
	Path    string `json:"path,omitempty"`
} // @name RequestUri

type RestApiRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	URI     RequestUri          `json:"uri"`
	Headers map[string][]string `json:"headers,omitempty"` // either map[string]string OR map[string][]string
	Params  interface{}         `json:"params,omitempty"`
	Body    interface{}         `json:"body,omitempty"`
	Timeout int64               `json:"timeout"`
	Caching int64               `json:"caching"`
} // @name RestApiRequest

type RestApiResponse struct {
	Request  RestApiRequest      `json:"request"`
	Code     int                 `json:"code"`
	Headers  map[string][]string `json:"headers,omitempty"`
	Body     interface{}         `json:"body,omitempty"`
	Runtime  *string             `json:"runtime,omitempty"`
	Datetime *string             `json:"datetime,omitempty"`
} // @name RestApiResponse

type StatusResponse struct {
	Code      *int         `json:"code,omitempty"`
	Data      *interface{} `json:"data,omitempty"`
	Message   *string      `json:"message,omitempty"`
	Path      *string      `json:"path,omitempty"`
	Runtime   *string      `json:"runtime,omitempty"`
	StatusUrl *string      `json:"statusUrl,omitempty"`
	TotalRows *int64       `json:"totalRows,omitempty"`
	DocsUrl   *string      `json:"docsUrl,omitempty"`
	Datetime  *string      `json:"datetime,omitempty"`
} // @name StatusResponse

type VersionResponse struct {
	Version string `json:"version"`
} // @name VersionResponse

type TtsVoicesList struct {
	Voices []TtsVoice `json:"voices"`
} // @name TtsVoicesList

type MapTtsVoicesList struct {
	Voices map[string]TtsVoice `json:"voices"`
} // @name MapTtsVoicesList

type TtsVoice struct {
	LanguageCodes []string `json:"languageCodes"`
	Name          string   `json:"name"`
	SsmlGender    string   `json:"ssmlGender"`
	SampleRateHz  int32    `json:"naturalSampleRateHertz"`
} // @name TtsVoice

type NewTtsRequest struct {
	Text     string  `json:"text,omitempty"`
	Language string  `json:"language,omitempty"` // e.g. "en-US"
	Voice    string  `json:"voice,omitempty"`    // e.g., "en-US-Wavenet-D"
	Gender   string  `json:"gender,omitempty"`   // e.g., "MALE", "FEMALE", "NEUTRAL"
	Pitch    float64 `json:"pitch,omitempty"`    // Range: -20.0 to 20.0, Default 0.00
	Speed    float64 `json:"speed,omitempty"`    // Range: 0.25 to 4.0, Default 1.00
	Volume   float64 `json:"volume,omitempty"`   // Range: -96.0 to 16.0, Default 0.00
} // @name NewTtsRequest

type NewTtsResponse struct {
	Success  bool   `json:"success"`
	Action   string `json:"action,omitempty"`
	Filename string `json:"filename,omitempty"`
	FileUrl  string `json:"url,omitempty"`
	Runtime  string `json:"runtime"`
} // @name NewTtsResponse

type PubSubRequest struct {
	Message      PubSubMessage `json:"message"`
	Subscription string        `json:"subscription,omitempty"`
} // @name PubSubRequest

type PubSubMessage struct {
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Data        string                 `json:"data,omitempty"`
	MessageId   string                 `json:"message_id,omitempty"`
	PublishTime string                 `json:"publish_time,omitempty"`
} // @name PubSubMessage

// /////// Redis Requests and Responses
type RedisDelRequest struct {
	Key string `json:"key"`
} // @name RedisDelRequest

type RedisDelResponse struct {
	Key     string  `json:"key"`
	Success bool    `json:"success"`
	Runtime *string `json:"runtime,omitempty"`
} // @name RedisDelResponse

type RedisGetRequest struct {
	Key string `json:"key"`
} // @name RedisGetRequest

type RedisGetResponse struct {
	Key     string       `json:"key"`
	Value   *interface{} `json:"value,omitempty"`
	Success bool         `json:"success"`
	Runtime *string      `json:"runtime,omitempty"`
} // @name RedisGetResponse

type RedisIncrRequest struct {
	Key      string `json:"key"`
	Value    int64  `json:"value"`
	Expires  int64  `json:"expires"`
	DoExpire bool   `json:"do_expire"`
} // @name RedisIncrRequest

type RedisIncrResponse struct {
	Rx      *RedisIncrRequest `json:"rx,omitempty"`
	Value   int64             `json:"value,omitempty"`
	Success bool              `json:"success"`
	Runtime *string           `json:"runtime,omitempty"`
} // @name RedisIncrResponse

type RedisKeysRequest struct {
	Pattern string `json:"pattern,omitempty"`
} // @name RedisKeysRequest

type RedisKeysResponse struct {
	Rx      *RedisKeysRequest `json:"rx,omitempty"`
	Data    *interface{}      `json:"data,omitempty"`
	Success bool              `json:"success"`
	Runtime *string           `json:"runtime,omitempty"`
} // @name RedisKeysResponse

type RedisSetRequest struct {
	Key     string       `json:"key"`
	Value   *interface{} `json:"value,omitempty"`
	Expires int64        `json:"expires"`
} // @name RedisSetRequest

type RedisSetResponse struct {
	Rx      *RedisSetRequest `json:"rx,omitempty"`
	Success bool             `json:"success"`
	Runtime *string          `json:"runtime,omitempty"`
} // @name RedisSetResponse

type RedisMSetRequest struct {
	Data []RedisSetRequest `json:"data,omitempty"`
} // @name RedisMSetRequest

type RedisMSetResponse struct {
	Data    []RedisSetResponse `json:"data,omitempty"`
	Success bool               `json:"success"`
	Runtime *string            `json:"runtime,omitempty"`
} // @name RedisMSetResponse

type GoogleTokenInfo struct {
	Alg           string `json:"alg,omitempty"`
	Aud           string `json:"aud,omitempty"`
	Azp           string `json:"azp,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified string `json:"email_verified,omitempty"`
	Exp           string `json:"exp,omitempty"`
	Iat           string `json:"iat,omitempty"`
	Iss           string `json:"iss,omitempty"`
	Kid           string `json:"kid,omitempty"`
	Sub           string `json:"sub,omitempty"`
	Typ           string `json:"typ,omitempty"`
} // @name GoogleTokenInfo

// Structs needed for httptimeout
type TimeoutTransport struct {
	http.Transport
	RoundTripTimeout time.Duration
} // @name TimeoutTransport

type respAndErr struct {
	resp *http.Response
	err  error
} // @name respAndErr

type netTimeoutError struct {
	error
} // @name netTimeoutError

type MirrorRequest struct {
	Attributes *map[string]string `json:"attributes,omitempty"`
	Data       *[]interface{}     `json:"data,omitempty"`
	Code       *int               `json:"code,omitempty"`
	// other example fields
} // @name MirrorRequest

///////////////////////////////////////////////////////////////////////////////////
//////// Global Shared Standalone API Functions Definition
///////////////////////////////////////////////////////////////////////////////////

// MirrorRequest Struct Methods
func (mr *MirrorRequest) SetCode(i int) *MirrorRequest {
	mr.Code = &i
	return mr
} // end method MirrorRequest.SetCode

func (mr MirrorRequest) GetCode() int {
	if mr.Code != nil {
		return *mr.Code
	}
	return 0
} // end method MirrorRequest.GetCode

// RedisDelResponse Struct Methods
func (rd *RedisDelResponse) SetRuntime(rt any) *RedisDelResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rd.Runtime = nil
		return rd
	}
	if str != "" {
		rd.Runtime = &str
	}
	return rd
} // end method RedisDelResponse.SetRuntime

func (rd RedisDelResponse) GetRuntime() string {
	if rd.Runtime != nil {
		return *rd.Runtime
	}
	return ""
} // end method RedisDelResponse.GetRuntime

// RedisGetResponse Struct Methods
func (rd *RedisGetResponse) SetRuntime(rt any) *RedisGetResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rd.Runtime = nil
		return rd
	}
	if str != "" {
		rd.Runtime = &str
	}
	return rd
} // end method RedisGetResponse.SetRuntime

func (rd RedisGetResponse) GetRuntime() string {
	if rd.Runtime != nil {
		return *rd.Runtime
	}
	return ""
} // end method RedisGetResponse.GetRuntime

func (rd *RedisGetResponse) SetValue(value any) *RedisGetResponse {
	rd.Value = &value
	return rd
} // end method RedisGetResponse.SetValue

func (rd *RedisGetResponse) GetValue() (v interface{}) {
	if rd.Value != nil {
		return *rd.Value
	}
	return v
} // end method RedisGetResponse.GetValue

// RedisIncrResponse Struct Methods
func (rd *RedisIncrResponse) SetRuntime(rt any) *RedisIncrResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rd.Runtime = nil
		return rd
	}
	if str != "" {
		rd.Runtime = &str
	}
	return rd
} // end method RedisIncrResponse.SetRuntime

func (rd RedisIncrResponse) GetRuntime() string {
	if rd.Runtime != nil {
		return *rd.Runtime
	}
	return ""
} // end method RedisIncrResponse.GetRuntime

func (rd *RedisIncrResponse) SetRx(rx RedisIncrRequest) *RedisIncrResponse {
	rd.Rx = &rx
	return rd
} // end method RedisIncrResponse.SetRx

func (rd *RedisIncrResponse) GetRx() (rx RedisIncrRequest) {
	if rd.Rx != nil {
		return *rd.Rx
	}
	return rx
} // end method RedisIncrResponse.GetRx

// RedisKeysResponse Struct Methods
func (rd *RedisKeysResponse) SetRuntime(rt any) *RedisKeysResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rd.Runtime = nil
		return rd
	}
	if str != "" {
		rd.Runtime = &str
	}
	return rd
} // end method RedisKeysResponse.SetRuntime

func (rd RedisKeysResponse) GetRuntime() string {
	if rd.Runtime != nil {
		return *rd.Runtime
	}
	return ""
} // end method RedisKeysResponse.GetRuntime

func (rd *RedisKeysResponse) SetRx(rx RedisKeysRequest) *RedisKeysResponse {
	rd.Rx = &rx
	return rd
} // end method RedisKeysResponse.SetRx

func (rd *RedisKeysResponse) GetRx() (rx RedisKeysRequest) {
	if rd.Rx != nil {
		return *rd.Rx
	}
	return rx
} // end method RedisKeysResponse.GetRx

// RedisSetRequest Struct Methods
func (rx *RedisSetRequest) SetValue(v any) *RedisSetRequest {
	rx.Value = &v
	return rx
} // end method RedisSetRequest.SetValue

func (rx *RedisSetRequest) GetValue() (v interface{}) {
	if rx.Value != nil {
		return *rx.Value
	}
	return v
} // end method RedisKeysResponse.GetValue

// RedisSetResponse Struct Methods
func (rd *RedisSetResponse) SetRuntime(rt any) *RedisSetResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rd.Runtime = nil
		return rd
	}
	if str != "" {
		rd.Runtime = &str
	}
	return rd
} // end method RedisSetResponse.SetRuntime

func (rd RedisSetResponse) GetRuntime() string {
	if rd.Runtime != nil {
		return *rd.Runtime
	}
	return ""
} // end method RedisSetResponse.GetRuntime

func (rd *RedisSetResponse) SetRx(rx RedisSetRequest) *RedisSetResponse {
	rd.Rx = &rx
	return rd
} // end method RedisSetResponse.SetRx

func (rd *RedisSetResponse) GetRx() (rx RedisSetRequest) {
	if rd.Rx != nil {
		return *rd.Rx
	}
	return rx
} // end method RedisSetResponse.GetRx

// RedisMSetResponse Struct Methods
func (rd *RedisMSetResponse) SetRuntime(rt any) *RedisMSetResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rd.Runtime = nil
		return rd
	}
	if str != "" {
		rd.Runtime = &str
	}
	return rd
} // end method RedisMSetResponse.SetRuntime

func (rd RedisMSetResponse) GetRuntime() string {
	if rd.Runtime != nil {
		return *rd.Runtime
	}
	return ""
} // end method RedisMSetResponse.GetRuntime

// RestApiResponse Struct Methods
func (rr *RestApiResponse) SetCode(i int) *RestApiResponse {
	//rr.Code = &i
	rr.Code = i
	return rr
} // end method RestApiResponse.SetCode

func (rr RestApiResponse) GetCode() int {
	return rr.Code
	//if rr.Code != nil {
	//    return *rr.Code
	//}
	//return 0
} // end method RestApiResponse.GetCode

func (rr *RestApiResponse) SetRuntime(rt any) *RestApiResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rr.Runtime = nil
		return rr
	}
	if str != "" {
		rr.Runtime = &str
	}
	return rr
} // end method RestApiResponse.SetRuntime

func (rr RestApiResponse) GetRuntime() string {
	if rr.Runtime != nil {
		return *rr.Runtime
	}
	return ""
} // end method RestApiResponse.GetRuntime

func (rr *RestApiResponse) SetDatetime(rt any) *RestApiResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = s("%+v", TimeDatetimeTz(rv))
	case int64:
		str = s("%+v", TimeDatetimeTz(UnixToTime(rv)))
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		rr.Datetime = nil
		return rr
	}
	if str != "" {
		rr.Datetime = &str
	}
	return rr
} // end method RestApiResponse.SetDatetime

func (rr RestApiResponse) GetDatetime() string {
	if rr.Datetime != nil {
		return *rr.Datetime
	}
	return ""
} // end method RestApiResponse.GetDatetime

// StatusResponse Struct Methods
func (sr *StatusResponse) SetCode(i int) *StatusResponse {
	sr.Code = &i
	return sr
} // end method StatusResponse.SetCode

func (sr StatusResponse) GetCode() int {
	if sr.Code != nil {
		return *sr.Code
	}
	return 0
} // end method StatusResponse.GetCode

func (sr *StatusResponse) SetMessage(m string) *StatusResponse {
	sr.Message = &m
	return sr
} // end method StatusResponse.SetMessage

func (sr *StatusResponse) UnsetMessage() *StatusResponse {
	sr.Message = nil
	return sr
} // end method StatusResponse.UnsetMessage

func (sr *StatusResponse) AppendMessage(m string) *StatusResponse {
	if sr.Message != nil {
		m = s("%s%s", sr.GetMessage(), m)
	}
	sr.SetMessage(m)
	return sr
} // end method StatusResponse.AppendMessage

func (sr StatusResponse) GetMessage() string {
	if sr.Message != nil {
		return *sr.Message
	}
	return ""
} // end method StatusResponse.GetMessage

func (sr *StatusResponse) SetData(datar ...any) *StatusResponse {
	if len(datar) == 0 {
		return sr
	}
	ds := make([]interface{}, 0)
	for ri := 0; ri < len(datar); ri++ {
		v := reflect.ValueOf(datar[ri])
		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			sliceLen := 0
			if v.Type().Kind() == reflect.Slice && v.Len() > 0 {
				sliceLen = v.Len()
			} else if v.Type().Kind() == reflect.Array && v.Type().Len() > 0 {
				sliceLen = v.Type().Len()
			}
			if sliceLen > 0 {
				for i := 0; i < sliceLen; i++ {
					ds = append(ds, v.Index(i).Interface())
				}
			}
		default:
			ds = append(ds, v.Interface())
		}
	} // end foreach datar
	dsi := interface{}(ds)
	sr.Data = &dsi
	return sr
} // end method StatusResponse.SetData

func (sr *StatusResponse) AppendData(datar ...any) *StatusResponse {
	if len(datar) == 0 {
		return sr
	}
	ds := make([]interface{}, 0)
	if sr.Data != nil {
		ds = sr.GetData()
	} // end sr.Data
	ds = append(ds, datar...)
	sr.SetData(ds)
	return sr
} // end method StatusResponse.AppendData

func (sr StatusResponse) GetData() []interface{} {
	if sr.Data != nil {
		dsi := *sr.Data
		return dsi.([]interface{})
	}
	return make([]interface{}, 0)
} // end method StatusResponse.GetData

func (sr *StatusResponse) SetDatetime(rt any) *StatusResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = s("%+v", TimeDatetimeTz(rv))
	case int64:
		str = s("%+v", TimeDatetimeTz(UnixToTime(rv)))
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		sr.Datetime = nil
		return sr
	}
	if str != "" {
		sr.Datetime = &str
	}
	return sr
} // end method StatusResponse.SetDatetime

func (sr StatusResponse) GetDatetime() string {
	if sr.Datetime != nil {
		return *sr.Datetime
	}
	return ""
} // end method StatusResponse.GetDatetime

func (sr *StatusResponse) SetPath(m string) *StatusResponse {
	sr.Path = &m
	return sr
} // end method StatusResponse.SetPath

func (sr StatusResponse) GetPath() string {
	if sr.Path != nil {
		return *sr.Path
	}
	return ""
} // end method StatusResponse.GetPath

func (sr *StatusResponse) SetRuntime(rt any) *StatusResponse {
	str := ""
	switch rv := rt.(type) {
	case string:
		str = rv
	case time.Time:
		str = Runtime(rv)
	case time.Duration:
		str = RuntimeDuration(rv)
	case float64:
		str = RuntimeFloat64(rv)
	default:
		e("Value type is not supported %T: %+v", rv, rv)
		sr.Runtime = nil
		return sr
	}
	if str != "" {
		sr.Runtime = &str
	}
	return sr
} // end method StatusResponse.SetRuntime

func (sr StatusResponse) GetRuntime() string {
	if sr.Runtime != nil {
		return *sr.Runtime
	}
	return ""
} // end method StatusResponse.GetRuntime

func (sr *StatusResponse) SetStatusUrl(m string) *StatusResponse {
	sr.StatusUrl = &m
	return sr
} // end method StatusResponse.SetStatusUrl

func (sr StatusResponse) GetStatusUrl() string {
	if sr.StatusUrl != nil {
		return *sr.StatusUrl
	}
	return ""
} // end method StatusResponse.GetStatusUrl

func (sr *StatusResponse) SetDocsUrl(m string) *StatusResponse {
	sr.DocsUrl = &m
	return sr
} // end method StatusResponse.SetDocsUrl

func (sr StatusResponse) GetDocsUrl() string {
	if sr.DocsUrl != nil {
		return *sr.DocsUrl
	}
	return ""
} // end method StatusResponse.GetDocsUrl

func (sr *StatusResponse) SetTotalRows(i int64) *StatusResponse {
	sr.TotalRows = &i
	return sr
} // end method StatusResponse.SetTotalRows

func (sr StatusResponse) GetTotalRows() int64 {
	if sr.TotalRows != nil {
		return *sr.TotalRows
	}
	return int64(0)
} // end method StatusResponse.GetTotalRows

// Part of the httptimeout
func (ne netTimeoutError) Timeout() bool { return true }

// If you don't set RoundTrip on TimeoutTransport, this will default to 5s
func (t *TimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.RoundTripTimeout.String() == "0s" {
		t.RoundTripTimeout = 5 * time.Second
	}
	timeout := time.After(t.RoundTripTimeout)
	resp := make(chan respAndErr, 1)

	go func() {
		r, e := t.Transport.RoundTrip(req)
		resp <- respAndErr{
			resp: r,
			err:  e,
		}
	}()

	select {
	case <-timeout: // A round trip timeout has occurred.
		t.Transport.CancelRequest(req)
		return nil, netTimeoutError{
			error: er("timed out after %s", t.RoundTripTimeout),
		}
	case r := <-resp: // Success!
		return r.resp, r.err
	}
} // end func TimeoutTransport.RoundTrip

// RegexpHandler
func (h *RegexpHandler) Handler(match string, handler http.Handler) {
	// Force all patterns to be absolute
	if SubStr(match, 0, 1) != "^" {
		match = s("^%s", match)
	}
	if SubStr(match, -1, 1) != "$" {
		match = s("%s$", match)
	}
	pattern, _ := regexp.Compile(match)
	h.routes = append(h.routes, &route{pattern, handler})
} // end RegexpHandler func Handler

func (h *RegexpHandler) HandleFunc(match string, handler func(http.ResponseWriter, *http.Request)) {
	pattern, _ := regexp.Compile(match)
	h.routes = append(h.routes, &route{pattern, http.HandlerFunc(handler)})
} // end RegexpHandler func HandleFunc

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//o("Received request path: %s with headers %+v", r.URL.Path, r.Header)
	for _, route := range h.routes {
		matchPathMethod := s("%s:%s", r.Method, r.URL.Path)
		matchPathAny := s("ANY:%s", r.URL.Path)
		//o("Checking Route Pattern: %s -- Request Path: %s || %s || %s", route.pattern, r.URL.Path, matchPathMethod, matchPathAny)
		if route.pattern.MatchString(r.URL.Path) || route.pattern.MatchString(matchPathMethod) || route.pattern.MatchString(matchPathAny) {
			//o("Route Pattern matched: %s", route.pattern)
			route.handler.ServeHTTP(w, r)
			return
		}
	}

	// Route Not Found
	o("No Route Patterns matched: %s", r.URL.Path)
	NotFoundHandler(w, r)
	return
} // end RegexpHandler func ServeHTTP

// Nil values init
func MakeNilError() error      { return nil }       // end func MakeNilError
func MakeNilReader() io.Reader { return nil }       // end func MakeNilReader
func MakeRedisNil() error      { return redis.Nil } // end func MakeRedisNil

func IsNil[T any](val T) bool {
	vf := reflect.ValueOf(val)
	for vf.IsValid() && (vf.Kind() == reflect.Interface || (vf.Kind() == reflect.Ptr && !vf.IsNil())) {
		vf = vf.Elem() // Get the inner value of an interface or pointer
	}
	if !vf.IsValid() {
		return true
	}
	switch vf.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		if vf.IsNil() {
			return true
		}
	}
	return false
} // end func IsNil

func IsEmpty[T any](val T) bool {
	if IsNil(val) {
		return true
	} // we'll count nil as empty
	vf := reflect.ValueOf(val)
	for vf.IsValid() && (vf.Kind() == reflect.Interface || (vf.Kind() == reflect.Ptr && !vf.IsNil())) {
		vf = vf.Elem() // Get the inner value of an interface or pointer
	}
	if !vf.IsValid() {
		return true
	}
	switch vf.Kind() {
	case reflect.Slice:
		if vf.Len() == 0 {
			return true
		}
	case reflect.Array:
		if vf.Type().Len() == 0 {
			return true
		}
	case reflect.String:
		if vf.String() == "" {
			return true
		}
	} // end switch type
	return false
} // end func IsEmpty

func IsEqual[A any, B any](a A, b B) (equal bool) {
	equal = false
	if reflect.DeepEqual(a, b) {
		// They are the same type... but DeepEqual doesn't do a good job of comparing maps deeply, lets check deeper
		equal = IsEqualInterface(interface{}(a), interface{}(b))
		//} else {
		//o("Not reflect.DeepEqual(a, b) => %T:%T", a, b)
		//o("Not reflect.DeepEqual(a, b), a: %+v", a)
		//o("Not reflect.DeepEqual(a, b), b: %+v", b)
	}
	return
} // end func IsEqual

func IsEqualInterface(a, b interface{}) (equal bool) {
	equal = false
	if a == b {
		return true
	}
	if s("%T", a) != s("%T", b) {
		//o("Not equal types a:b => %T:%T", a, b)
		return
	}
	switch av := a.(type) {
	case map[string]interface{}:
		switch bv := b.(type) {
		case map[string]interface{}:
			equal = IsEqualMapStringInterface(av, bv)
		default:
			return
		} // end switch b
	case []interface{}:
		switch bv := b.(type) {
		case []interface{}:
			equal = IsEqualSliceInterface(av, bv)
		default:
			return
		} // end switch b
	default:
		avf := reflect.ValueOf(a)
		for avf.IsValid() && (avf.Kind() == reflect.Interface || (avf.Kind() == reflect.Ptr && !avf.IsNil())) {
			avf = avf.Elem() // Get the inner value of an interface or pointer
		}
		bvf := reflect.ValueOf(b)
		for bvf.IsValid() && (bvf.Kind() == reflect.Interface || (bvf.Kind() == reflect.Ptr && !bvf.IsNil())) {
			bvf = bvf.Elem() // Get the inner value of an interface or pointer
		}
		if !avf.IsValid() && !bvf.IsValid() {
			return true
		} // both not valid
		if !avf.IsValid() {
			return false
		}
		if !bvf.IsValid() {
			return false
		}

		avt := reflect.TypeOf(avf)
		avk := avt.Kind()
		bvt := reflect.TypeOf(bvf)
		bvk := bvt.Kind()
		if avk != bvk {
			return false
		}

		if avk == reflect.Struct || avk == reflect.Slice || avk == reflect.Map || avk == reflect.Array {
			avj := JsonEncodeStrOut(a)
			bvj := JsonEncodeStrOut(b)
			if avj == bvj {
				return true
			}
		} else {
			avb := AnyToByte(a)
			bvb := AnyToByte(b)
			if bytes.Equal(avb, bvb) {
				return true
			}
		}
		if avf == bvf {
			return true
		}
	} // end switch a
	return
} // end func IsEqualInterface

func IsEqualMapStringInterface(a, b map[string]interface{}) (equal bool) {
	equal = false
	avj := JsonEncodeStrOut(a)
	bvj := JsonEncodeStrOut(b)
	if avj == bvj {
		return true
	}
	return
} // end func IsEqualMapStringInterface

func IsEqualSliceInterface(a, b []interface{}) (equal bool) {
	equal = false
	avj := JsonEncodeStrOut(a)
	bvj := JsonEncodeStrOut(b)
	if avj == bvj {
		return true
	}
	return
} // end func IsEqualSliceInterface

// GetEnvVar / * requires a key parameter to search for
// from an env file and sets path to the dot env file
// finally returns the value retrieved from the environment file
func GetEnvVar(key string, opt ...string) string {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// opt = slice of string inputs: defaultValue, flagKey, envKey, flagType
	envFilePath := ".env"
	defaultValue := ""
	envKey := key
	flagKey := ""
	flagType := "string"
	if len(opt) >= 1 {
		if opt[0] != "" {
			defaultValue = opt[0]
		}
		if len(opt) >= 2 {
			if opt[1] != "" {
				envFilePath = opt[1]
			}
			if len(opt) >= 3 {
				if opt[2] != "" {
					flagKey = opt[2]
				}
				if len(opt) >= 4 {
					if opt[3] != "" {
						envKey = opt[3]
					}
					if len(opt) >= 5 {
						if opt[4] != "" {
							flagType = opt[4]
						}
					}
				} // end if opt 3
			} // end if opt 2
		} // end if opt 1
	} // end if opt 0
	value := defaultValue
	inputDefaultValue := defaultValue
	if envKey != "" {
		envValue, envFound := os.LookupEnv(envKey)
		//o("envKey: %s, envFound: %+v, envValue: %s", envKey, envFound, envValue)
		if envFound {
			if envValue != "" {
				defaultValue = envValue
				value = defaultValue
			}
		}
	}

	envFilePathFound := false
	if key != "" && envFilePath != "" {
		exists, _, err := PathExists(envFilePath)
		if exists == true && err == nil {
			envFilePathFound = true
		}
		//o("key: %s, envFilePath: %s, envFilePathFound: %+v", key, envFilePath, envFilePathFound)
	}
	if envFilePathFound == true {
		viper.SetConfigFile(envFilePath)

		err := viper.ReadInConfig()
		if err != nil {
			e("Couldn't read config file: %+v", err)
			return value //in case it was found in the os.LookupEnv() call
		}
		cnfValue := viper.Get(key) //can return nil, not always interface{}
		//o("cnfValue: %+v", cnfValue)
		cnfValueStr := ""
		if cnfValue != nil {
			cnfValueStr = cnfValue.(string)
			if cnfValueStr != "" {
				value = cnfValueStr
			}
		}
	}

	//o("flagKey: %s, flagType: %s, value: %+v", flagKey, flagType, value)
	if value != "" {
		defaultValue = value
	}
	wasFlagValueFound := false
	if flagKey != "" {
		switch flagType {
		case "int":
			defaultValueInt, err := strconv.ParseInt(defaultValue, 10, 64)
			if err != nil {
				e("Failed to Parse Int from defaultValue %s", defaultValue)
			}
			flagValue, flagValueFound := GetCmdArg(flagKey, flagType, defaultValueInt)
			wasFlagValueFound = flagValueFound
			if flagValueFound == true {
				flagValueStr := strconv.FormatInt(flagValue.(int64), 10)
				//o("flagKey: %s (%s) = flagValue: %+v (%T) : defaultValue: %+v (%T)", flagKey, flagType, flagValue, flagValue, defaultValueInt, defaultValueInt)
				if flagValueStr != "" {
					value = flagValueStr
				}
			}
		case "bool":
			defaultValueBool := false
			if defaultValue == "true" || defaultValue == "TRUE" || defaultValue == "1" {
				defaultValueBool = true
			}
			flagValue, flagValueFound := GetCmdArg(flagKey, flagType, defaultValueBool)
			wasFlagValueFound = flagValueFound
			if flagValueFound == true {
				o("flagKey: %s (%s) = flagValue: %+v (%T) : defaultValue: %+v (%T)", flagKey, flagType, flagValue, flagValue, defaultValueBool, defaultValueBool)
				if flagValue.(bool) == true {
					value = "true"
				} else {
					value = "false"
				}
			}
		default:
			// Else String
			flagValue, flagValueFound := GetCmdArg(flagKey, flagType, defaultValue)
			wasFlagValueFound = flagValueFound
			if flagValueFound == true {
				//o("flagKey: %s (%s) = flagValue: %+v (%T) : defaultValue: %+v (%T)", flagKey, flagType, flagValue, flagValue, defaultValue, defaultValue)
				value = flagValue.(string)
			}
		}
		//o("flagValue: %+v", value)
	}
	if value == "" && wasFlagValueFound == false {
		value = inputDefaultValue
	}
	return value
} // end func GetEnvVar

// This is just an alias for GetEnvVar()
func GetEnvVarStr(key string, opt ...string) string {
	return GetEnvVar(key, opt...)
} // end func GetEnvVarStr

func GetEnvVarInt64(key string, def, min, max int64, opt ...string) int64 {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	valueStr := strconv.FormatInt(def, 10)
	opt = append([]string{valueStr}, opt...)
	valueStr = GetEnvVar(key, opt...)
	value := ParseIntDefMinMax(valueStr, def, min, max)
	return value
} // end func GetEnvVarInt64

func GetEnvVarBool(key string, def bool, opt ...string) bool {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	valueStr := BoolToStr(def)
	opt = append([]string{valueStr}, opt...)
	valueStr = GetEnvVar(key, opt...)
	value := AnyToBool(valueStr, def)
	return value
} // end func GetEnvVarBool

func GetCmdArg(argKey, argType string, value any) (any, bool) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	//argKeyLen := len(argKey)
	found := false
	if len(os.Args) > 0 {
		for i, v := range os.Args {
			thisKey := v
			thisValue := ""
			thisValueSet := false
			nextValue := ""
			nextValueFound := false
			if len(os.Args) > i+1 {
				nextValue := os.Args[i+1]
				if nextValue != "" {
					nextValueFound = true
					if SubStr(nextValue, 0, 1) == "-" {
						nextValue = ""
						nextValueFound = false
					} // end if -
				} // end if nextValue
			} // end if len os.Args
			if strings.Contains(v, "=") == true {
				vv := strings.Split(v, "=")
				thisKey = vv[0]
				thisValue = vv[1]
				thisValueSet = true
			} // end if =
			if thisKey == argKey || thisKey == "-"+argKey || thisKey == "--"+argKey {
				if thisValueSet == false && nextValueFound == true {
					thisValue = nextValue
					thisValueSet = nextValueFound
				}
				found = true
				//o("GetCmdArg: %s (%s) thisValue: %+v (%T)", argKey, argType, thisValue, thisValue)
				switch argType {
				case "int":
					if thisValue != "" {
						thisValueInt, err := strconv.ParseInt(thisValue, 10, 64)
						if err != nil {
							e("GetCmdArg: %s (%s) got error strconv.ParseInt: %+v", argKey, argType, err)
						} else {
							value = thisValueInt
						}
					}
				case "bool":
					value = true
					if thisValue == "false" || thisValue == "FALSE" || thisValue == "0" || thisValue == "nil" {
						value = false
					}
				default:
					// else string
					value = thisValue
				} // end switch argType
			} // end if argKey
		} // end for range Args
	} // end if Args
	return value, found
} // end func GetCmdArg

func GetFileContentType(path string) (string, error) {
	if path == "" {
		return "", er("No Path Input Provided")
	}
	exists, isDirectory, err := PathExists(path)
	if err != nil || exists == false {
		return "", err
	}
	if isDirectory {
		return "text/directory", nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// to sniff the content type only the first
	// 512 bytes are used.
	buf := make([]byte, 512)

	_, err = file.Seek(0, io.SeekStart)
	if err != nil && err != io.EOF {
		return "", err
	}

	bytesRead, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Slice to remove fill-up zero values which cause a wrong content type detection in the next step
	buf = buf[:bytesRead]

	// the function that actually does the trick
	contentType := http.DetectContentType(buf)
	if contentType == "" {
		contentType = "text/plain"
	}
	return contentType, nil
} // end func GetFileContentType

func prepend[T any](newVal T, existing ...T) []T {
	r := make([]T, 0)
	r = append(r, newVal)
	r = append(r, existing...)
	return r
} // end func prepend

func InArray(val any, arrays ...any) (exists bool, input int, index int) {
	total := 0
	start := TimeNow()
	arraysn := len(arrays)
	if arraysn > 0 {
		vf := reflect.ValueOf(val)
		for vf.IsValid() && (vf.Kind() == reflect.Interface || (vf.Kind() == reflect.Ptr && !vf.IsNil())) {
			vf = vf.Elem() // Get the inner value of an interface or pointer
		}
		if !vf.IsValid() {
			return
		}
		valKind := vf.Kind()
		if valKind == reflect.String && StrContains(val.(string), ",") {
			valx := StrSplit(val.(string), ",")
			val = reflect.ValueOf(valx).Interface() // interface to []string
			valKind = reflect.TypeOf(val).Kind()
		}
		for i := 0; i < arraysn; i++ {
			array := arrays[i]
			t := reflect.TypeOf(array)
			switch t.Kind() {
			case reflect.Slice, reflect.Array:
				v := reflect.ValueOf(array)
				sliceLen := 0
				if t.Kind() == reflect.Slice && v.Len() > 0 {
					sliceLen = v.Len()
				} else if t.Kind() == reflect.Array && v.Type().Len() > 0 {
					sliceLen = v.Type().Len()
				}
				if sliceLen > 0 {
					for ii := 0; ii < sliceLen; ii++ {
						if valKind == reflect.Slice || valKind == reflect.Array {
							valx := val.([]string)
							for vi := 0; vi < len(valx); vi++ {
								if IsEqual(valx[vi], v.Index(ii).Interface()) {
									input = i
									index = ii
									exists = true
									if total > 100 {
										o("InArray Matched. Runtime %s.", Runtime(start))
									}
									return
								} // end if val
							}
						} else if IsEqual(val, v.Index(ii).Interface()) {
							input = i
							index = ii
							exists = true
							if total > 100 {
								o("InArray Matched. Runtime %s.", Runtime(start))
							}
							return
						} // end if val
						total++
					} // end foreach ValueOf(array)
				} // end if sliceLen
			case valKind:
				if IsEqual(val, array) {
					input = i
					index = 0
					exists = true
					if total > 100 {
						o("InArray Matched. Runtime %s.", Runtime(start))
					}
					return
				} // end if val
				total++
			default:
				total++
			} // end switch TypeOf(array)
		} // end foreach array
	} // end if arrays
	if total > 100 {
		o("InArray Matched. Runtime %s.", Runtime(start))
	}
	return
} // end func InArray

func InStringSlice(value string, checks ...any) (in bool, iv int) {
	in = false
	iv = 0
	if len(checks) == 0 {
		return
	}
	for i := 0; i < len(checks); i++ {
		check := checks[i]
		switch cv := check.(type) {
		case []string:
			if len(cv) > 0 {
				for ci := 0; ci < len(cv); ci++ {
					if value == cv[ci] {
						return true, ci
					} // end if value
				} // end foreach cv
			} // end if cv
		case string:
			if value == cv {
				return true, i
			} // end if value
		default:
			e("Input must be either []string or ...string, got (%T)", cv)
		} // end switch check
	} // end foreach checks
	return
} // end func InStringSlice

func CsvInStringSlice(value, sep string, check []string) (clean string, any bool, all bool) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	if value == "" || len(check) == 0 {
		return
	}
	cleanr := make([]string, 0)
	valuer := []string{value}
	if sep != "" && StrContains(value, sep) {
		valuer = StrSplit(value, sep)
	}
	for vi := 0; vi < len(valuer); vi++ {
		if in, ci := InStringSlice(valuer[vi], check); in {
			if inr, _ := InStringSlice(check[ci], cleanr); !inr {
				cleanr = append(cleanr, check[ci])
			} // end if in cleanr
			any = true
		} // end if in check
	} // end foreach valuer
	all = bool(len(cleanr) == len(valuer))
	if len(cleanr) == 1 {
		clean = cleanr[0]
	} else if len(cleanr) > 1 {
		if Trim(sep) == "" {
			sep = ","
		} // default clean csv
		clean = strings.Join(cleanr, Trim(sep)+" ")
	}
	return
} // end func CsvInStringSlice

func StrContains(a string, b string, opt ...bool) bool {
	// opt: slice of inputs:
	// insensitive, default true
	var insensitive bool = true
	if len(opt) > 0 {
		insensitive = opt[0]
	}
	if insensitive {
		return strings.Contains(
			strings.ToLower(a),
			strings.ToLower(b),
		)
	} else {
		return strings.Contains(a, b)
	}
} // end func StrContains

func StrReplace(str, o, n string) string {
	str = strings.ReplaceAll(str, o, n)
	return str
} // end func StrReplace

func StrSliceJoin(a []string, sep string) string {
	str := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(a)), sep), "[]")
	return str
} // end func StrSliceJoin

func StrSplit(input string, opt ...string) []string {
	var sep string
	var output []string
	input = strings.TrimSpace(input)
	if len(opt) > 0 {
		sep = opt[0]
	}
	if input != "" && sep != "" {
		output = strings.Split(input, sep)
		for i, str := range output {
			output[i] = strings.TrimSpace(str)
		}
	} else {
		output = []string{input}
	} // end if input
	return output
} // end func StrSplit

func Lo(str string) string {
	return strings.ToLower(str)
} // end func Lo

func Up(str string) string {
	return strings.ToUpper(str)
} // end func Up

func Trim(str string) string {
	return strings.TrimSpace(str)
} // end func Trim

// SubStr is a UTF-8 aware substring function
// Definition and functionality matches PHP substr
func SubStr(input string, opt ...int) string {
	if input == "" {
		return ""
	}
	asRunes := []rune(input)

	// opt = slice of inputs:
	// offset int -- default 0
	// length int -- default {length of input string}
	var offset, length int = 0, len(asRunes)
	if len(opt) >= 1 {
		offset = opt[0]
		if len(opt) >= 2 {
			length = opt[1]
		}
	}

	// Negative offset, count back from the end
	if offset < 0 {
		offset = len(asRunes) + offset
		if offset < 0 {
			// Still Negative offset, clamp to start of string
			offset = 0
		}
	}

	// Offset is over the string length
	if offset >= len(asRunes) {
		return ""
	}

	// Negative length, count back from the end,  - offset + (-length)
	if length < 0 {
		length = len(asRunes) - offset + length
	}

	// Make sure we don't over extend past the end
	if offset+length > len(asRunes) {
		length = len(asRunes) - offset
	}

	return string(asRunes[offset : offset+length])
} // end func SubStr

func DynamicPlaceholdersClean(body []byte) []byte {
	if len(body) == 0 {
		return body // No Body Provided
	}
	defer RecoverErrorStack(TraceFile())     // Panic Error handling wrapped
	keys := DynamicPlaceholderBodyKeys(body) // map[ReplaceValue] = KeyName
	if len(keys) > 0 {
		for replace, _ := range keys {
			body = bytes.ReplaceAll(body, []byte(replace), []byte(""))
		}
	} // end if keys
	return body
} // end func DynamicPlaceholdersClean

func DynamicPlaceholderBodyKeys(body []byte) map[string]string {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	keys := make(map[string]string)
	pattern := "\\[@([a-zA-Z0-9\\-\\_]+)(#[a-zA-Z0-9\\-\\_]+)?(#.*)?\\]"
	re, err := regexp.Compile(pattern)
	if err != nil {
		e("DynamicPlaceholderBodyKeys: Error compiling regular expression: %s", pattern)
		return keys
	}
	if !re.Match(body) {
		return keys // No more [@placeholders]
	}
	allBodyKeyMatches := re.FindAllSubmatch(body, -1)
	allBodyKeyMatchesLen := len(allBodyKeyMatches)
	if allBodyKeyMatchesLen > 0 {
		for ki := 0; ki < allBodyKeyMatchesLen; ki++ {
			var dataKey string = ""
			var replaceKey string = ""
			thisKeyMatches := allBodyKeyMatches[ki]
			//o("DynamicPlaceholderBodyKeys: thisKeyMatches: %+v", thisKeyMatches)
			if len(thisKeyMatches) > 0 {
				replaceKey = string(thisKeyMatches[0][:]) // the full string matching == [@someKey#function#variable]
				if len(thisKeyMatches) > 1 {
					dataKey = string(thisKeyMatches[1][:]) // the first (value) == someKey
				}
			}
			if replaceKey == "" {
				continue
			} // Our regexp matched [@]... just ignore it
			if dataKey == "" {
				dataKey = replaceKey
			} // This shouldn't happen... but just in case
			keys[replaceKey] = dataKey
		} // end foreach allBodyKeyMatches
	} // end if allBodyKeyMatchesLen
	return keys
} // end func DynamicPlaceholderBodyKeys

func DynamicPlaceholderKeyMatches(p, k string, matches map[string]string) (bool, string) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	if k == "" {
		return false, ""
	}
	if len(matches) == 0 {
		return false, ""
	}
	for r, m := range matches {
		if m == "" {
			continue
		}
		if m == k {
			return true, r
		} // exact match
		if p != "" {
			if m == p+"_"+k {
				return true, r
			} // exact match with prefix
			lpk := len(p + "_" + k)
			lm := len(m)
			if lm < lpk && SubStr(p+"_"+k, -1*lm) == m {
				return true, r
			} // exact sub prefix match
		} // end if prefix

		// try lowercase with no underscores or dashes
		mi := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(m, "_", ""), "-", ""))
		ki := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(k, "_", ""), "-", ""))
		if mi == ki {
			return true, r
		} // insensitive match
		if p != "" {
			pi := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(p, "_", ""), "-", ""))
			if mi == pi+ki {
				return true, r
			} // insensitive prefix match
			lpki := len(pi + ki)
			lmi := len(mi)
			if lmi < lpki && SubStr(pi+ki, -1*lmi) == mi {
				return true, r
			} // insensitive sub prefix match
		} // end if prefix
	} // end foreach matches m
	return false, ""
} // end func DynamicPlaceholderKeyMatches

// DynamicPlaceholdersPrefixBytes replaces all of the key values from the data in the body
func DynamicPlaceholdersPrefixBytes(body []byte, prefix string, clean bool, rdata ...any) []byte {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	if len(body) == 0 {
		return body // No Body Provided
	}
	if len(rdata) == 0 {
		if clean {
			body = DynamicPlaceholdersClean(body)
		}
		return body // No Data Provided
	}
	pattern := "\\[@([a-zA-Z0-9\\-\\_]+)(#[a-zA-Z0-9\\-\\_]+)?(#.*)?\\]"
	re, err := regexp.Compile(pattern)
	if err != nil {
		e("DynamicPlaceholdersPrefixBytes: Error compiling regular expression: %s", pattern)
		return body
	}
	if !re.Match(body) {
		return body // No [@placeholders] Found
	}
	bodyKeys := DynamicPlaceholderBodyKeys(body)
	for rdi, data := range rdata {
		if rdi > 0 {
			if !re.Match(body) {
				return body // No more [@placeholders]
			}
		}

		switch d := data.(type) {
		case []interface{}:
			for keyi := 0; keyi < len(d); keyi++ {
				if keyi > 0 {
					if !re.Match(body) {
						return body // No more [@placeholders]
					}
				}
				prefixKeyi := prefix
				if prefix != "" {
					prefixKeyi += s("_%d", keyi)
				} else {
					prefixKeyi = s("%d", keyi)
				}
				// step into the interface{}
				body = DynamicPlaceholdersPrefixBytes(body, prefixKeyi, false, d[keyi])
			} // end for i
		case map[string]interface{}:
			keyi := 0
			for key, val := range d {
				if keyi > 0 {
					if !re.Match(body) {
						return body // No more [@placeholders]
					}
				}
				prefixKeyi := prefix
				if prefixKeyi != "" {
					prefixKeyi += s("_%s", key)
				} else {
					prefixKeyi = key
				}
				switch v := val.(type) {
				case []byte:
					matches, replace := DynamicPlaceholderKeyMatches(prefixKeyi, key, bodyKeys)
					if matches {
						keyPlaceholder := []byte(replace)
						//o("DynamicPlaceholdersPrefixBytes: Replacing \"%s\" => \"%s\"", string(keyPlaceholder), string(byteVal))
						body = bytes.ReplaceAll(body, keyPlaceholder, v)
					} // end if matches
				case string, error, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
					matches, replace := DynamicPlaceholderKeyMatches(prefixKeyi, key, bodyKeys)
					if matches {
						byteVal := AnyToByte(v)
						keyPlaceholder := []byte(replace)
						//o("DynamicPlaceholdersPrefixBytes: Replacing \"%s\" => \"%s\"", string(keyPlaceholder), string(byteVal))
						body = bytes.ReplaceAll(body, keyPlaceholder, byteVal)
					} // end if matches
				default:
					// Struct, Map, Slice, Array, etc... just go deeper
					body = DynamicPlaceholdersPrefixBytes(body, prefixKeyi, false, v)
				} // end switch v type
				keyi++
			} // end for range d
		case map[string][]byte:
			keyi := 0
			for key, val := range d {
				if keyi > 0 {
					if !re.Match(body) {
						return body // No more [@placeholders]
					}
				}
				prefixKeyi := prefix
				if prefixKeyi != "" {
					prefixKeyi += s("_%s", key)
				} else {
					prefixKeyi = key
				}
				matches, replace := DynamicPlaceholderKeyMatches(prefixKeyi, key, bodyKeys)
				if matches {
					keyPlaceholder := []byte(replace)
					//o("DynamicPlaceholdersPrefixBytes: Replacing \"%s\" => \"%s\"", string(keyPlaceholder), string(byteVal))
					body = bytes.ReplaceAll(body, keyPlaceholder, val)
				} // end if matches
				keyi++
			} // end for range d
		case map[string]string:
			keyi := 0
			for key, val := range d {
				if keyi > 0 {
					if !re.Match(body) {
						return body // No more [@placeholders]
					}
				}
				prefixKeyi := prefix
				if prefixKeyi != "" {
					prefixKeyi += s("_%s", key)
				} else {
					prefixKeyi = key
				}
				matches, replace := DynamicPlaceholderKeyMatches(prefixKeyi, key, bodyKeys)
				if matches {
					byteVal := AnyToByte(val)
					keyPlaceholder := []byte(replace)
					//o("DynamicPlaceholdersPrefixBytes: Replacing \"%s\" => \"%s\"", string(keyPlaceholder), string(byteVal))
					body = bytes.ReplaceAll(body, keyPlaceholder, byteVal)
				} // end if matches
				keyi++
			} // end for range d
		default:
			vf := reflect.ValueOf(d)
			for vf.IsValid() && (vf.Kind() == reflect.Interface || (vf.Kind() == reflect.Ptr && !vf.IsNil())) {
				vf = vf.Elem() // Get the inner value of an interface or pointer
			}
			if !vf.IsValid() {
				continue
			}

			t := reflect.TypeOf(vf)
			switch t.Kind() {
			case reflect.Map:
				// Some other map than map[string]interface{}, convert to map[string]interface{}, try again
				vmap := make(map[string]interface{})
				err := MergeIntoMap(vmap, vf.Interface())
				if err != nil {
					e("Failed to Merge Map type (%T) into map[string]interface{}: %+v", d, err)
				} else {
					body = DynamicPlaceholdersPrefixBytes(body, prefix, false, vmap)
				}
			case reflect.Struct:
				v := StructToMapStringInterface(d)
				if len(v) > 0 {
					body = DynamicPlaceholdersPrefixBytes(body, prefix, false, v)
				}
			case reflect.Slice, reflect.Array:
				sliceLen := 0
				if t.Kind() == reflect.Slice && vf.Len() > 0 {
					sliceLen = vf.Len()
				} else if t.Kind() == reflect.Array && vf.Type().Len() > 0 {
					sliceLen = vf.Type().Len()
				}
				if sliceLen > 0 {
					for keyi := 0; keyi < sliceLen; keyi++ {
						if keyi > 0 {
							if !re.Match(body) {
								return body // No more [@placeholders]
							}
						}
						var prefixKey string = ""
						key := s("%d", keyi)
						if prefix != "" {
							prefixKey = prefix + "_" + key
						} else {
							prefixKey = key
						}

						// Interfaceify the value
						vvf := vf.Index(keyi).Interface()

						//o("DynamicPlaceholdersPrefixBytes: %s: %+v (%T)", key, value, value)
						switch vvv := vvf.(type) {
						case []byte:
							matches, replace := DynamicPlaceholderKeyMatches(prefixKey, key, bodyKeys)
							if matches {
								keyPlaceholder := []byte(replace)
								//o("DynamicPlaceholdersPrefixBytes: Replacing \"%s\" => \"%s\"", string(keyPlaceholder), string(byteVal))
								body = bytes.ReplaceAll(body, keyPlaceholder, vvv)
							} // end if matches
						case string, error, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
							matches, replace := DynamicPlaceholderKeyMatches(prefixKey, key, bodyKeys)
							if matches {
								byteVal := AnyToByte(vvv)
								keyPlaceholder := []byte(replace)
								//o("DynamicPlaceholdersPrefixBytes: Replacing \"%s\" => \"%s\"", string(keyPlaceholder), string(byteVal))
								body = bytes.ReplaceAll(body, keyPlaceholder, byteVal)
							} // end if matches
						default:
							// Struct, Map, Slice, Array, etc... just go deeper
							body = DynamicPlaceholdersPrefixBytes(body, prefixKey, false, vvv)
						} // end switch vvv type
					} // end foreach key value
				} // end if len data
			default:
				msi := make(map[string]interface{})
				v := AnyToByte(d)
				msi[prefix] = v
				body = DynamicPlaceholdersPrefixBytes(body, "", false, msi)
			} // end switch TypeOf(data).Kind()
		} // end switch data.(type)
	} // end foreach data
	if clean {
		body = DynamicPlaceholdersClean(body)
	}
	return body
} // end func DynamicPlaceholdersPrefixBytes

func DynamicPlaceholdersBytes(body []byte, clean bool, data ...any) []byte {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	body = DynamicPlaceholdersPrefixBytes(body, "", clean, data...)
	return body
} // end func DynamicPlaceholdersBytes

func DynamicPlaceholdersStr(body string, clean bool, data ...any) string {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	byteBody := []byte(body)
	byteBody = DynamicPlaceholdersPrefixBytes(byteBody, "", clean, data...)
	body = string(byteBody)
	return body
} // end func DynamicPlaceholdersStr

func DynamicPlaceholdersAny[T any](body *T, clean bool, data ...any) error {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var err error
	jBytes, err := JsonEncode(body)
	if err != nil {
		return err
	}
	jBytes = DynamicPlaceholdersPrefixBytes(jBytes, "", clean, data...)
	err = JsonDecode(jBytes, &body)
	return err
} // end func DynamicPlaceholdersAny

func IntToByte(v any) []byte {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var err error
	var intV int64 = 0
	switch vv := v.(type) {
	case int64:
		intV = vv
	case int32:
		intV = int64(vv)
	case int:
		intV = int64(vv)
	case float64:
		intV = int64(vv)
	case float32:
		intV = int64(vv)
	case string:
		intV, err = ParseInt(vv)
		if err != nil {
			e("Couldn't parse string input to int64, defaulting to 0 instead of %+v => %+v", vv, intV)
			intV = 0
		}
	default:
		e("Input type not supported %T defaulting to 0", vv)
	} // end switch
	strVal := strconv.FormatInt(intV, 10)
	byteVal := []byte(strVal)
	return byteVal
} // end func IntToByte

func AnyToByte(value any) []byte {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// Handle nil interfaces or pointers first
	t := reflect.TypeOf(value)
	switch t.Kind() {
	case reflect.Interface, reflect.Ptr:
		rv := reflect.ValueOf(value)
		for rv.IsValid() && (rv.Kind() == reflect.Interface || (rv.Kind() == reflect.Ptr && !rv.IsNil())) {
			rv = rv.Elem()
		}
		if !rv.IsValid() || ((rv.Kind() == reflect.Interface || rv.Kind() == reflect.Ptr) && rv.IsNil()) {
			return []byte("")
		}
	}

	// Check for other types
	switch v := value.(type) {
	case error:
		if v == nil {
			return []byte("")
		}
		str := s("%+v", v)
		return []byte(str)
	case []byte:
		return v
	case string:
		return []byte(v)
	case bool:
		if v == true {
			return []byte("true")
		} else {
			return []byte("false")
		}
	case int:
		return IntToByte(v)
	case int8:
		return IntToByte(v)
	case int16:
		return IntToByte(v)
	case int32:
		return IntToByte(v)
	case uint:
		return IntToByte(v)
	case uint8:
		return IntToByte(v)
	case uint16:
		return IntToByte(v)
	case uint32:
		return IntToByte(v)
	case uint64:
		return IntToByte(v)
	case int64:
		strVal := strconv.FormatInt(v, 10)
		return []byte(strVal)
	case float32:
		strVal := strconv.FormatFloat(float64(v), 'f', -1, 32)
		return []byte(strVal)
	case float64:
		strVal := strconv.FormatFloat(v, 'f', -1, 64)
		return []byte(strVal)
	case map[string]interface{}:
		jsonByte, err := JsonEncode(v)
		if err == nil {
			return jsonByte
		} else {
			e("Failed to JSON encode the %T: %+v", v, v)
			e("%+v", err)
		}
	case interface{}:
		switch vv := v.(type) {
		default:
			o("Value was an interface of %T: Trying to Marshal into JSON", vv)
			t := reflect.TypeOf(vv)
			if t.Kind() == reflect.Struct || t.Kind() == reflect.Slice || t.Kind() == reflect.Map || t.Kind() == reflect.Array {
				jsonByte, err := JsonEncode(vv)
				if err == nil {
					return jsonByte
				} else {
					e("Failed to JSON encode the %s: %+v (%T): %+v", t.Kind().String(), vv, vv, err)
					e("%+v", err)
				}
			} else {
				// Step into the interface and try again
				rv := reflect.ValueOf(vv)
				for rv.IsValid() && (rv.Kind() == reflect.Interface || (rv.Kind() == reflect.Ptr && !rv.IsNil())) {
					rv = rv.Elem()
				}
				if rv.IsValid() {
					switch rv.Kind() {
					case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
						jsonByte, err := JsonEncode(vv)
						if err == nil {
							return jsonByte
						} else {
							e("Failed to JSON encode the %s: %+v (%T): %+v", t.Kind().String(), vv, vv, err)
							e("%+v", err)
						}
					case reflect.String:
						return AnyToByte(rv.String())
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						return AnyToByte(rv.Int())
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
						return AnyToByte(rv.Uint())
					case reflect.Float32, reflect.Float64:
						return AnyToByte(rv.Float())
					default:
						if rv.CanInterface() {
							return AnyToByte(rv.Interface())
						} else {
							e("Type not supported %T resolved to %s: %+v", vv, rv.Kind().String(), vv)
						}
					}
				} // end if rv.IsValid
			} // end if Struct
		} // end switch type v
	default:
		t := reflect.TypeOf(value)
		switch t.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			jsonByte, err := JsonEncode(v)
			if err == nil {
				return jsonByte
			} else {
				e("Failed to JSON encode the Struct: %+v (%T): %+v", v, v, err)
				e("%+v", err)
			}

		default:
			e("Data Value type (%T) is not supported yet: %+v", v, v)
		}
	} // end switch type value
	return []byte("")
} // end func AnyToByte

func AnyToBool(value any, opt ...bool) bool {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// Optional Default Param
	var def bool = false
	if len(opt) > 0 {
		def = opt[0]
	}
	var r bool = def
	if value == nil {
		r = false
		return r
	}
	switch v := value.(type) {
	case bool:
		r = v
	case string:
		if v == "" {
			return r
		}
		trues := []string{"1", "true", "TRUE", "yes", "YES", "Y", "y"}
		falses := []string{"0", "false", "FALSE", "nil", "NULL", "n", "N", "no", "NO"}
		if exists, _ := InStringSlice(v, trues); exists {
			r = true
		} else if exists, _ := InStringSlice(v, falses); exists {
			r = false
		} else {
			e("String Value %s is not in %+v or %+v... using default: %+v", trues, falses, def)
		}
	case int:
		if v == 1 {
			r = true
		} else if v == 0 {
			r = false
		} else {
			e("Int Value %d is not 1 or 0... using default: %+v", v, def)
		}
	case int32:
		if v == 1 {
			r = true
		} else if v == 0 {
			r = false
		} else {
			e("Int32 Value %d is not 1 or 0... using default: %+v", v, def)
		}
	case int64:
		if v == 1 {
			r = true
		} else if v == 0 {
			r = false
		} else {
			e("Int64 Value %d is not 1 or 0... using default: %+v", v, def)
		}
	case float32:
		if v == 1 {
			r = true
		} else if v == 0 {
			r = false
		} else {
			e("Float32 Value %d is not 1 or 0... using default: %+v", v, def)
		}
	case float64:
		if v == 1 {
			r = true
		} else if v == 0 {
			r = false
		} else {
			e("Float64 Value %d is not 1 or 0... using default: %+v", v, def)
		}
	case error:
		if v == nil {
			r = false
		} else {
			r = true
		}
	default:
		e("Value type %T is not supported, using default: %+v", v, def)
	} // end switch value type
	return r
} // end func AnyToBool

func BoolToInt(value bool) int {
	if value {
		return 1
	} else {
		return 0
	}
} // end func BoolToInt

func BoolToStr(value bool, opt ...string) string {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var format string = "1"
	if len(opt) > 0 {
		format = opt[0]
	}
	r := ""
	test := [][]string{
		[]string{"0", "1"},
		[]string{"false", "true"},
		[]string{"FALSE", "TRUE"},
		[]string{"no", "yes"},
		[]string{"NO", "YES"},
	}
	i := BoolToInt(value)
	for _, t := range test {
		if exists, _ := InStringSlice(format, t); exists {
			return t[i]
		}
	} // end foreach test
	return r
} // end func BoolToStr

func AnyToHeadersMap(input any) (headers map[string][]string, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	headers = make(map[string][]string)
	err = nil
	switch in := input.(type) {
	case interface{}:
		switch i := in.(type) {
		case http.Header: // underlying is map[string][]string
			//o("Interface of type http.Header: %+v", i)
			//headers = i
			for key, vals := range i {
				for hi := 0; hi < len(vals); hi++ {
					_, kok := headers[key]
					if kok {
						headers[key] = append(headers[key], vals[hi])
					} else {
						headers[key] = []string{vals[hi]}
					}
				}
			}
			//o("headers: %+v", headers)
		case map[string]string:
			//headers = i
			for key, val := range i {
				_, kok := headers[key]
				if kok {
					headers[key] = append(headers[key], val)
				} else {
					headers[key] = []string{val}
				}
			}
		case map[string][]string:
			//headers = i
			for key, vals := range i {
				for hi := 0; hi < len(vals); hi++ {
					_, kok := headers[key]
					if kok {
						headers[key] = append(headers[key], vals[hi])
					} else {
						headers[key] = []string{vals[hi]}
					}
				}
			}
		default:
			err = eer("Input inner Interface Type (%T) not supported for Headers: %+v", i, i)
		}

	case http.Header: // underlying is map[string][]string
		//o("Directly type http.Header")
		//headers = i
		for key, vals := range in {
			for hi := 0; hi < len(vals); hi++ {
				_, kok := headers[key]
				if kok {
					headers[key] = append(headers[key], vals[hi])
				} else {
					headers[key] = []string{vals[hi]}
				}
			}
		}
		//o("headers: %+v", headers)

	case map[string]string:
		//headers = i
		for key, val := range in {
			_, kok := headers[key]
			if kok {
				headers[key] = append(headers[key], val)
			} else {
				headers[key] = []string{val}
			}
		}

	case map[string][]string:
		//headers = i
		for key, vals := range in {
			for hi := 0; hi < len(vals); hi++ {
				_, kok := headers[key]
				if kok {
					headers[key] = append(headers[key], vals[hi])
				} else {
					headers[key] = []string{vals[hi]}
				}
			}
		}

	default:
		err = eer("Interface Type (%T) not supported for Headers: %+v", in, in)
	}
	if err != nil {
		return
	}

	//o("headers: %+v", headers)
	copyMap := make(map[string][]string)
	copyBytes, err := JsonEncode(headers)
	if err != nil {
		return
	}
	err = JsonDecode(copyBytes, &copyMap)
	return copyMap, err
} // end func AnyToHeadersMap

func StructNilStringPointers[T any](p *T) *T {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// Set any empty string pointer to nil
	r := reflect.ValueOf(p)
	for r.IsValid() && (r.Kind() == reflect.Interface || (r.Kind() == reflect.Ptr && !r.IsNil())) {
		r = r.Elem() // Get the inner value of an interface or pointer
	}
	if !r.IsValid() {
		return p
	}
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i) // the field value as a reflect.Value
		if f.Kind() == reflect.Ptr && f.Type().Elem().Kind() == reflect.String && !f.IsNil() && f.Elem().String() == "" {
			f.Set(reflect.Zero(f.Type())) // set the trx field value pointer to nil
		} // end if not nil empty string pointer
	} // end foreach field
	return p
} // end func StructNilStringPointers

func StructToMapStringInterface(data any) map[string]interface{} {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	result := make(map[string]interface{})
	switch d := data.(type) {
	case map[string]interface{}:
		return d

	default:
		t := reflect.TypeOf(d) // Struct definition
		if t.Kind() == reflect.Struct {
			StructNilStringPointers(&d)
			v := reflect.ValueOf(d) // This Struct's value as reflect.Value
			sn := reflect.TypeOf(d).Name()
			for i := 0; i < t.NumField(); i++ {
				omitempty := false
				f := t.Field(i) // Struct definition of field
				fn := f.Name    // Default if no json name
				jkey, ok := f.Tag.Lookup("json")
				key := jkey
				if ok && key != "" {
					//o("Json Key found for %s.%s: %s", sn, fn, jkey)
					if key != "-" && StrContains(key, ",omitempty") {
						omitempty = true
						keyx := StrSplit(key, ",")
						key = keyx[0]
					}
					//o("Json Key found for %s.%s: %s -> %s (omitempty %+v)", sn, fn, jkey, key, omitempty)
				}
				if key == "-" {
					continue
				}
				if !ok || key == "" || jkey == "" {
					key = fn
				}
				if key != "" && key != "-" {
					vf := v.Field(i) // This Struct Field
					for vf.IsValid() && (vf.Kind() == reflect.Interface || (vf.Kind() == reflect.Ptr && !vf.IsNil())) {
						vf = vf.Elem() // Get the inner value of an interface or pointer
					}
					skipField := false
					if omitempty && vf.IsValid() {
						switch vf.Kind() {
						case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
							if vf.IsNil() {
								skipField = true
								continue
							}
						case reflect.String:
							if vf.String() == "" {
								skipField = true
								continue
							}
						} // end switch vf.Kind
					} // end if omitempty
					isEmpty := true
					vfKind := vf.Kind().String()
					rfInts := []string{
						reflect.Int.String(), reflect.Int8.String(), reflect.Int16.String(), reflect.Int32.String(), reflect.Int64.String(),
						reflect.Uint.String(), reflect.Uint8.String(), reflect.Uint16.String(), reflect.Uint32.String(), reflect.Uint64.String(),
					}
					isIntType, _, _ := InArray(vfKind, rfInts)
					if !skipField && vf.IsValid() && !isIntType {
						isEmpty = vf.IsZero()
					}
					if jkey != "" && skipField {
						o("JSON Key found for %s.%s: %s -> %s (omitempty %+v => %+v)", sn, fn, jkey, key, omitempty, isEmpty)
					}
					if !skipField && vf.CanInterface() {
						if omitempty && isEmpty && !isIntType {
							continue
						} else {
							switch vv := vf.Interface().(type) {
							default:
								result[key] = vv // as the value of the interface
							}
						} // end if isEmpty
					} // end if vf.CanInterface
				} // end if ok key
			} // end for t.NumField
		} // end if t.Kind
	} // end switch x.type
	return result
} // end func StructToMapStringInterface

// Create a SHA256 hash of the byte data
func HashStr(data any) string {
	dataBytes := AnyToByte(data)
	hashBytes := sha256.Sum256(dataBytes)
	hashStr := hex.EncodeToString(hashBytes[:])
	return hashStr
} // end func HashStr

// Just a wrapper to reduce imports duplication
func ParseInt(str string) (i int64, err error) {
	i, err = strconv.ParseInt(str, 10, 64)
	return
} // end func ParseInt

func ParseIntDefMinMax(str string, def int64, min int64, max int64) int64 {
	r := def
	if str == "" {
		return r
	}
	p, err := ParseInt(str)
	if err == nil {
		r = p
		if r < min {
			r = min
		} else if r > max {
			r = max
		}
	} // end if err
	return r
} // end func ParseIntDefMinMax

func ParseFloat(str string, bitSize int) (f float64, err error) {
	f, err = strconv.ParseFloat(str, bitSize)
	return
} // end func ParseFloat

// Cryptographically safe random int64 between min/max (positive or negative)
func RandomInt(min int64, max int64) int64 {
	if min == max {
		return min
	}
	if min > max {
		// Min/Max were reversed, flip them
		tmp_max := min
		min = max
		max = tmp_max
	}

	// Shift the base to 0
	diff_min_max := max - min // these are all regular int64

	// Random 0 - N
	big_rand, err := crand.Int(crand.Reader, big.NewInt(diff_min_max)) // Input is math/big.Int, Output is math/big.Int
	if err != nil {
		e("Failed to Generate random integer between 0 - %d for RandomInt(%d, %d)", diff_min_max, min, max)
		return min
	}

	// convert back to regular int64
	diff_rand := big_rand.Int64()

	// Shift base back to min
	randInt := diff_rand + min

	return randInt
} // end func RandomInt

func IsNaN(f float64) bool {
	return math.IsNaN(f)
} // end func IsNaN

func Round(f float64) float64 {
	if math.IsNaN(f) {
		return 0
	}
	return math.Round(f)
} // end func Round

func Float64ToInt64(f float64) int64 {
	if math.IsNaN(f) {
		return 0
	}
	return int64(math.Round(f))
} // end func Float64ToInt64

func InterfaceToInt64(a any, opt ...int64) int64 {
	return AnyToInt64(a, opt...)
} // end func InterfaceToInt64

func AnyToInt64(a any, opt ...int64) int64 {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var d int64 = 0
	if len(opt) > 0 {
		d = opt[0]
	}
	var r int64 = d
	switch i := a.(type) {
	case int:
		if !math.IsNaN(float64(i)) {
			r = int64(i)
		}
	case int32:
		if !math.IsNaN(float64(i)) {
			r = int64(i)
		}
	case int64:
		if !math.IsNaN(float64(i)) {
			r = i
		}
	case uint:
		if !math.IsNaN(float64(i)) {
			r = int64(i)
		}
	case uint32:
		if !math.IsNaN(float64(i)) {
			r = int64(i)
		}
	case uint64:
		if !math.IsNaN(float64(i)) {
			r = int64(i)
		}
	case float32:
		if !math.IsNaN(float64(i)) {
			r = int64(math.Round(float64(i)))
		}
	case float64:
		if !math.IsNaN(i) {
			r = int64(math.Round(i))
		}
	case string:
		v, err := strconv.ParseInt(i, 10, 64)
		if err != nil {
			e("%+v", err)
		}
		if !math.IsNaN(float64(v)) {
			r = v
		}
	default:
		e("Input type is not supported! %T: %+v", i, i)
	} // end switch i
	if math.IsNaN(float64(r)) {
		r = d
	}
	return r
} // end func AnyToInt64

func AnyToFloat64(a any, opt ...float64) float64 {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var d float64 = 0
	if len(opt) > 0 {
		d = opt[0]
	}
	var r float64 = d
	switch i := a.(type) {
	case int:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case int32:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case int64:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case uint:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case uint32:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case uint64:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case float32:
		if !math.IsNaN(float64(i)) {
			r = float64(i)
		}
	case float64:
		if !math.IsNaN(i) {
			r = i
		}
	case string:
		v, err := strconv.ParseFloat(i, 64)
		if err != nil {
			e("%+v", err)
		}
		if !math.IsNaN(v) {
			r = v
		}
	default:
		e("Input type is not supported! %T: %+v", i, i)
	} // end switch i
	if math.IsNaN(r) {
		r = d
	}
	return r
} // end func AnyToFloat64

func AnyMapToMapStringInterface[T any](input T) (output map[string]interface{}) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	output = make(map[string]interface{})
	t := reflect.TypeOf(input)
	switch t.Kind() {
	case reflect.Map:
		ii := reflect.ValueOf(input).Interface()
		err := MergeIntoMap(output, ii)
		if err != nil {
			e("Failed to Merge Map of type (%T) into map[string]interface{}: %+v", input, err)
		}
	case reflect.Slice, reflect.Array:
		vf := reflect.ValueOf(input)
		sliceLen := 0
		if t.Kind() == reflect.Slice && vf.Len() > 0 {
			sliceLen = vf.Len()
		} else if t.Kind() == reflect.Array && vf.Type().Len() > 0 {
			sliceLen = vf.Type().Len()
		}
		if sliceLen > 0 {
			for keyi := 0; keyi < sliceLen; keyi++ {
				// Stringify the key
				key := s("%d", keyi)

				// Interfaceify the value
				vi := vf.Index(keyi).Interface()

				output[key] = vi
			} // end for keyi in slice
		} // end if sliceLen
	} // end switch input
	return
} // end func AnyMapToMapStringInterface

func LoadTimezone(opt ...string) bool {
	var timezoneEnvKey string
	if len(opt) > 0 {
		timezoneEnvKey = opt[0]
	}
	if timezoneEnvKey == "" {
		timezoneEnvKey = "API_TIMEZONE"
	}
	now := time.Now()
	nowTz := s("%+v", now.Location().String())
	defaultApiTimezone = GetEnvVar("TZ", nowTz, ".env", "TZ", "TZ", "string")
	thisApiTimezone = GetEnvVar(timezoneEnvKey, defaultApiTimezone, ".env", timezoneEnvKey, timezoneEnvKey, "string")
	if thisApiTimezone == nowTz || thisApiTimezone == "" {
		return true
	}
	o("Loading Timezone %s", thisApiTimezone)
	loc, err := time.LoadLocation(thisApiTimezone)
	if err != nil {
		e("Couldn't load %s: %+v", thisApiTimezone, err)
		if thisApiTimezone == defaultApiTimezone {
			return false
		} else if defaultApiTimezone == nowTz || defaultApiTimezone == "" {
			return true
		}
		loc, err = time.LoadLocation(defaultApiTimezone)
		if err != nil {
			e("Couldn't load %s: %+v", defaultApiTimezone, err)
			if defaultApiTimezone == "UTC" {
				return false
			} else if nowTz == "UTC" {
				return true
			}
			loc, err = time.LoadLocation("UTC")
			if err != nil {
				return false
			}
		}
	}
	time.Local = loc // -> this is setting the global timezone
	return true
} // end func LoadTimezone

func Today() time.Time {
	// time.Time of {today} at 00:00:00.00000
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return today
} // end func Today

func TodayStr() string {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	today := Today().Format("2006-01-02")
	return today
} // end func TodayStr

func NowDatetime() string { return Datetime() } // alias function
func Datetime(opt ...time.Time) string {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	t := time.Now()
	if len(opt) > 0 {
		t = opt[0]
	}
	dt := t.Format("2006-01-02 15:04:05")
	return dt
} // end func Datetime

func NowDatetimeTz() string { return DatetimeTz() } // alias function
func DatetimeTz(opt ...time.Time) string {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	t := time.Now()
	if len(opt) > 0 {
		t = opt[0]
	}
	dt := t.Format("2006-01-02 15:04:05 MST")
	return dt
} // end func Datetime

func DatetimeParse(str string) time.Time {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	t, _ := TimeParse("2006-01-02 15:04:05", str)
	return t
} // end func DatetimeParse

func ParseDatetime(str string) time.Time {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	return DatetimeParse(str) // Just an alias function
} // end func ParseDatetime

func ParseDuration(d time.Duration) int64 {
	i := int64(d / time.Second)
	return i
} // end func ParseDuration

func TimeAdd(t time.Time, d time.Duration) time.Time {
	n := t.Add(d)
	return n
} // end func TimeAdd

func TimeSub(t time.Time, d time.Duration) time.Time {
	n := t.Add(-d)
	return n
} // end func TimeSub

func TimeDatetime(t time.Time) string {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	dt := t.Format("2006-01-02 15:04:05")
	return dt
} // end func TimeDatetime

func TimeDatetimeTz(t time.Time) string {
	// Layout = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	dt := t.Format("2006-01-02 15:04:05 MST")
	return dt
} // end func TimeDatetimeTz

func TimeDiff(t1 time.Time, t2 time.Time) int64 {
	var d int64 = 0
	var diff time.Duration
	if t1.Before(t2) {
		diff = t2.Sub(t1)
	} else {
		diff = t1.Sub(t2)
	}
	d = int64(diff.Seconds())
	return d
} // end func TimeDiff

func TimeDuration(i int64) time.Duration {
	d := time.Duration(i) * time.Second
	return d
} // end func TimeDuration

func TimeDurationDiff(d1 time.Duration, d2 time.Duration) int64 {
	var d int64 = 0
	if d1 > d2 {
		d = int64(d1.Seconds() - d2.Seconds())
	} else {
		d = int64(d2.Seconds() - d1.Seconds())
	}
	return d
} // end func TimeDurationDiff

func TimeHttpHeaderStr(t time.Time) string {
	// Last-Modified: <day-name>, <day> <month> <year> <hour>:<minute>:<second> GMT
	// Last-Modified: Mon, 02 Jan 2006 15:04:05 GMT
	dt := t.UTC().Format(http.TimeFormat)
	return dt
} // end func TimeHttpHeaderStr

func TimeNear(t1, t2 time.Time, sec int64) bool {
	if TimeDiff(t1, t2) <= sec {
		return true
	}
	return false
} // end func TimeNear

func TimeNow() time.Time {
	return time.Now()
} // end func TimeNow

func TimeParse(f, str string) (time.Time, error) {
	t := time.Time{}
	if f != "" && str != "" {
		tp, err := time.Parse(f, str)
		if err != nil {
			e("TimeParse: Str: '%s', Format: '%s', Error: %+v", str, f, err)
		} else {
			return tp, nil
		}
	}
	return t, nil
} // end func TimeParse

func TimeSince(t time.Time) time.Duration {
	d := time.Since(t)
	return d
} // end func TimeSince

// Sleep in Seconds
func sleep(i int64) { TimeSleep(i) } // Alias Function
func TimeSleep(i int64) {
	time.Sleep(time.Duration(i) * time.Second)
} // end func TimeSleep

// Sleep in Milliseconds
func msleep(i int64) { TimeSleepMs(i) } // Alias Function
func TimeSleepMs(i int64) {
	time.Sleep(time.Duration(i) * time.Millisecond)
} // end func TimeSleepMs

// Sleep in Microseconds
func usleep(i int64) { TimeUsleep(i) } // Alias Function
func TimeUsleep(i int64) {
	time.Sleep(time.Duration(i) * time.Microsecond)
} // end func TimeUsleep

func TimeUnix(opt ...time.Time) int64 {
	t := time.Now()
	if len(opt) > 0 {
		t = opt[0]
	}
	return t.Unix()
} // end func TimeUnix

func TimeUnixMicro(opt ...time.Time) int64 {
	t := time.Now()
	if len(opt) > 0 {
		t = opt[0]
	}
	return t.UnixMicro()
} // end func TimeUnixMicro

func TimeZero() time.Time {
	return time.Time{}
} // end func TimeZero

func RuntimeDurationFloat(start time.Time, duration time.Duration) float64 {
	runtime := time.Since(start)
	result := float64(runtime / duration)
	return result
} // end func RuntimeDurationFloat

func RuntimeDuration(d time.Duration) string {
	result := d.Seconds()
	resultStr := RuntimeFloat64(float64(result))
	return resultStr
} // end func RuntimeDuration

func RuntimeFloat64(f float64) string {
	resultStr := s("%.5fs", f)
	return resultStr
} // end func RuntimeFloat64

func RuntimeSecondsStr(start time.Time) string {
	return Runtime(start)
} // end func RuntimeSecondsStr

func Runtime(start time.Time) string {
	runtime := time.Since(start)
	result := runtime.Seconds()
	resultStr := RuntimeFloat64(float64(result))
	return resultStr
} // end func Runtime

func IsRuntimeStr(str string) bool {
	isRuntime := false
	if len(str) > 1 {
		if SubStr(str, -1) == "s" {
			str = SubStr(str, 0, -1)
			_, err := ParseFloat(str, 64)
			if err == nil {
				isRuntime = true
			}
		}
	}
	return isRuntime
} // end func IsRuntimeStr

func UnixToTime(i int64) time.Time {
	now := time.Now()
	uxt := time.Unix(i, 0).In(now.Location())
	return uxt
} // end func UnixToTime

func InDeniedPaths(path string) bool {
	if path != "" && path != "/" {
		// TODO TODO -- These should be defined in an ENV var
		if StrContains(path, ".env") == true || StrContains(path, ".htaccess") || StrContains(path, "../") {
			o("Path %s is not allowed!", path)
			return true
		}
	}
	//o("Path %s is allowed", path)
	return false
} // end func InDeniedPaths

func PathBasename(path string) string {
	return filepath.Base(path)
} // end func PathBasename

// PathExists returns whether the given file or directory exists
//
//	Input
//	    Path string
//	Return
//	    FileExists bool
//	    IsDirectory bool
//	    Error error
func PathExists(path string) (bool, bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return true, true, nil
		} else {
			return true, false, nil
		}
	}
	if os.IsNotExist(err) {
		return false, false, nil
	}
	return false, false, err
} // end func PathExists
/*
// PathWritable checks if a path is writable (local disk only)
func PathWritable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
} // end func PathWritable
*/
// PathJoin joins any number of strings together separated by single / (even if the strings have them as well)
func PathJoin(paths ...string) string {
	full := ""
	if len(paths) > 0 {
		for _, path := range paths {
			if len(path) > 0 {
				if len(full) > 0 {
					if SubStr(full, -1) == "/" {
						full = SubStr(full, 0, -1)
					} // Remove trailing /
					if SubStr(path, 0, 1) == "/" {
						path = SubStr(path, 1)
					} // Remove leading /
					full += "/" + path
				} else {
					full = path
				} // end if full
			} // end if baseDir
		} // end foreach path
	} // end if paths
	return full
} // end func PathJoin

func ReadFileBytes(path string) (fileBytes []byte, found bool, isDirectory bool, foundName string, size int64, updated time.Time, mime string, err error) {
	fileBytes = make([]byte, 0)
	if path == "" {
		err = er("No Path Input Provided")
		return
	}
	found, isDirectory, err = PathExists(path)
	if err != nil {
		return
	} else if found == false {
		err = er("File Not Found: %s", path)
		return
	} else if isDirectory {
		err = er("Path is a Directory: %s", path)
		return
	}

	// Get the Contact MIME Type
	mime, err = GetFileContentType(path)
	if err != nil {
		return
	}

	// Open the file handle
	file, err := os.Open(path)
	if err != nil {
		e("Failed to Open the file %s: %+v", path, err)
		return
	}
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		// File exists but is not readable!
		e("Failed to Stat the file %s: %+v", path, err)
		return
	}
	found = true
	foundName = path

	// Read the file into a byte slice
	updated = stat.ModTime()
	ssize := stat.Size()
	fileBytes = make([]byte, ssize)
	sizeb, err := bufio.NewReader(file).Read(fileBytes)
	size = int64(sizeb)
	if sizeb == 0 || (err != nil && err != io.EOF) {
		if err != nil {
			e("Failed to Read the file %s: %+v", path, err)
		} else if sizeb == 0 {
			e("File is empty! %s", path)
		}
		return
	}
	// stat.Size() is not always the Read() size! Trim the fileBytes slice to the Read size
	if ssize != size {
		fileBytes = fileBytes[:size]
	}
	return
} // end func ReadFileBytes

func LogErr(message string) string {
	if SubStr(message, 0, 7) != "" {
		message = "Error" + ": " + message
	}
	return message
} // end func LogErr

func LogFunc(message string) string {
	tbFunc := TraceFunc()
	lenTbFunc := len(tbFunc)
	if lenTbFunc > 0 && tbFunc != "main" {
		if SubStr(message, 0, lenTbFunc+1) != tbFunc+":" {
			message = tbFunc + ": " + message
		}
	}
	return message
} // end func LogFunc

// Add the Log Format to a Message String
func LogMsg(message string, intr ...interface{}) string {
	message = LogFunc(message)
	message = s(message, intr...)
	if message == "" {
		return ""
	}
	if thisHostname != "" && thisHostname != "localhost" {
		if !StrContains(message, s("[%s]  ", thisHostname)) {
			message = s("[%s]  %s", thisHostname, message)
		}
	}
	message = s("[%s]  %s\n", time.Now().Format("2006-01-02 15:04:05.000000"), message)
	return message
} // end func LogMsg

// Sprintf Message String
func s(message string, intr ...interface{}) string {
	if message != "" || len(intr) > 0 {
		message = fmt.Sprintf(message, intr...)
	}
	return message
} // end func s

// Errorf Message String
func er(message string, intr ...interface{}) error {
	var err error
	message = s(message, intr...)
	if message == "" {
		return nil
	}
	err = fmt.Errorf(message)
	return err
} // end func er

// Error Message String and print to STD ERR
func eer(message string, intr ...interface{}) error {
	message = s(message, intr...)
	if message == "" {
		return nil
	}
	err := er(message)
	e("%+v", err)
	return err
} // end func eer

// Message String and print to STD ERR
func erm(message string, intr ...interface{}) string {
	message = s(message, intr...)
	if message == "" {
		return message
	}
	e(message)
	return message
} // end func erm

// Message String and print to STD OUT
func oom(message string, intr ...interface{}) string {
	message = s(message, intr...)
	if message == "" {
		return message
	}
	o(message)
	return message
} // end func oom

// Log Formatted Output / Standard Out
func d(str string, i ...interface{}) { o(str, i...) } // Alias function
func o(message string, intr ...interface{}) {
	if message == "" {
		return
	}
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.ReplaceAll(message, `\r\n`, "\n")
	message = strings.ReplaceAll(message, `\n`, "\n")
	messages := strings.Split(message, "\n")
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		msg = s(msg, intr...)
		if msg != "" {
			msg = LogMsg(msg)
			fmt.Fprintf(os.Stdout, msg)
		}
	}
} // end func o

// Log Formatted Error / Standard Err
func e(message string, intr ...interface{}) {
	if message == "" {
		return
	}
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.ReplaceAll(message, `\r\n`, "\n")
	message = strings.ReplaceAll(message, `\n`, "\n")
	messages := strings.Split(message, "\n")
	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		msg = s(msg, intr...)
		if msg != "" {
			msg = LogErr(msg)
			msg = LogMsg(msg)
			fmt.Fprintf(os.Stderr, msg)
		}
	}
} // end func e

// Unformatted Print Message
func p(message string, intr ...interface{}) {
	if message == "" {
		return
	}
	message = s(message, intr...)
	fmt.Printf(message)
} // end func p

func NewUuid() string {
	uid := uuid.New()
	id := uid.String()
	return id
} // end func NewUuid

func GetAcnsApiUrl(r *http.Request, api_path string) string {
	var acnsApiUrl string = ""
	if thisAcnsApiDomain != "" {
		acnsApiUrl = thisAcnsApiDomain
	} else {
		acnsApiUrl = GetRequestBaseUrl(r)
	} // end if thisAcnsApiDomain
	if acnsApiUrl == "" {
		return ""
	}
	_, domain, _, _, err := ParseRawURL(acnsApiUrl)
	if err != nil {
		return ""
	}
	if domain == "" {
		return ""
	}
	thisAcnsApiDomain = domain // set the global
	acnsApiUrl = "https://" + domain
	acnsApiUrl = PathJoin(acnsApiUrl, api_path)
	return acnsApiUrl
} // end func GetAcnsApiUrl

func NewAcnsApiRequest[T any](r *http.Request, method string, url string, response *T, opt ...interface{}) (RestApiResponse, error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	if SubStr(url, 0, 5) != "http:" && SubStr(url, 0, 6) != "https:" && SubStr(url, 0, 4) != "wss:" {
		// URL was given as just the path without domain
		url = GetAcnsApiUrl(r, url)
	}

	// Set the default headers
	headers := make(map[string][]string)

	apiAuthHeaderSet := false
	apiCmdHeaderSet := false
	apiKeyHeaderSet := false
	acceptHeaderSet := false
	contentTypeHeaderSet := false

	// Read the optional inputs
	acns_api_cmd := ""
	acns_opt := make([]interface{}, 0)
	if len(opt) > 0 {
		// Optional: body (string OR []byte OR io.Reader), data (map[string]interface{}), headers (map[string]string), timeout (time.Duration OR int64)
		for _, opt := range opt {
			switch input := opt.(type) {
			case string:
				acns_api_cmd = input
			case http.Header: // underlying is map[string][]string
				for key, vals := range input {
					if ok, _ := InStringSlice(key, []string{"Accept", "accept"}); ok {
						acceptHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"Content-Type", "content-type"}); ok {
						contentTypeHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"x-api-key", "X-Api-Key"}); ok {
						apiKeyHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"x-api-cmd", "X-Api-Cmd"}); ok {
						apiCmdHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"Authorization", "authorization", "Authentication", "authentication"}); ok {
						apiAuthHeaderSet = true
					}

					for hi := 0; hi < len(vals); hi++ {
						_, kok := headers[key]
						if kok {
							headers[key] = append(headers[key], vals[hi])
						} else {
							headers[key] = []string{vals[hi]}
						}
					}
				}
			case map[string]string:
				for key, val := range input {
					if ok, _ := InStringSlice(key, []string{"Accept", "accept"}); ok {
						acceptHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"Content-Type", "content-type"}); ok {
						contentTypeHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"x-api-key", "X-Api-Key"}); ok {
						apiKeyHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"x-api-cmd", "X-Api-Cmd"}); ok {
						apiCmdHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"Authorization", "authorization", "Authentication", "authentication"}); ok {
						apiAuthHeaderSet = true
					}

					_, kok := headers[key]
					if kok {
						headers[key] = append(headers[key], val)
					} else {
						headers[key] = []string{val}
					}
				}
			case map[string][]string:
				for key, vals := range input {
					if ok, _ := InStringSlice(key, []string{"Accept", "accept"}); ok {
						acceptHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"Content-Type", "content-type"}); ok {
						contentTypeHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"x-api-key", "X-Api-Key"}); ok {
						apiKeyHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"x-api-cmd", "X-Api-Cmd"}); ok {
						apiCmdHeaderSet = true
					} else if ok, _ := InStringSlice(key, []string{"Authorization", "authorization", "Authentication", "authentication"}); ok {
						apiAuthHeaderSet = true
					}

					for hi := 0; hi < len(vals); hi++ {
						_, kok := headers[key]
						if kok {
							headers[key] = append(headers[key], vals[hi])
						} else {
							headers[key] = []string{vals[hi]}
						}
					}
				}
			default:
				acns_opt = append(acns_opt, input)
			} // end switch input
		} // end for opt
	} // end if opt
	if !apiCmdHeaderSet && acns_api_cmd != "" {
		headers["x-api-cmd"] = []string{acns_api_cmd}
	}
	if !apiKeyHeaderSet && !apiAuthHeaderSet {
		headers["x-api-key"] = []string{thisAcnsApiKey}
	}
	if !acceptHeaderSet {
		headers["Accept"] = []string{"application/json"}
	}
	if !contentTypeHeaderSet {
		headers["Content-Type"] = []string{"application/json"}
	}

	acns_opt = prepend(interface{}(headers), acns_opt...) // prepend
	return NewRestApiRequest(method, url, response, acns_opt...)
} // end func NewAcnsApiRequest

func ParseRawURL(rawurl string) (scheme, domain, port, path string, err error) {
	u, err := url.ParseRequestURI(rawurl)
	if err != nil || u.Host == "" {
		if SubStr(rawurl, 0, 5) != "http:" && SubStr(rawurl, 0, 6) != "https:" && SubStr(rawurl, 0, 4) != "wss:" {
			rawurl = "https://" + rawurl
		}
		u, repErr := url.ParseRequestURI(rawurl)
		if repErr != nil {
			e("ParseRawURL: Could not parse raw url: %s, error: %+v", rawurl, err)
			return
		}
		scheme = ""
		domain = u.Hostname()
		port = u.Port()
		err = nil
		return
	} // end if Host

	scheme = u.Scheme
	domain = u.Hostname()
	port = u.Port()
	path = u.Path
	//o("scheme: %s, domain: %s, port: %s", scheme, domain, port)
	return
} // end func ParseRawURL

func CheckIfTcpPortAvailable(port string) (available bool, err error) {
	available = false
	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil {
		// Port is not an integer string
		err = er("Invalid Port %q: %+v", port, err)
		return
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		// Port is not available
		err = nil
		return
	}

	err = ln.Close()
	if err != nil {
		// Couldn't close the listener for some reason
		err = er("Test Listener Failed to Close on Port %q: %+v", port, err)
		return
	}

	available = true
	err = nil
	return
} // end func CheckIfTcpPortAvailable

func CheckIpIsGcpLoadBalancer(ip string) bool {
	// HC: 35.191.0.0/16, 130.211.0.0/22
	// GFE: 34.96.0.0/20, 34.127.192.0/18, 130.211.0.0/22, 35.191.0.0/16
	ip4 := net.ParseIP(ip)
	if ip4 == nil || ip4.IsPrivate() || ip4.IsUnspecified() || ip4.IsLoopback() {
		return false
	}
	ip = s("%s", ip4)
	knownGcpCIDRs := []string{
		"34.64.0.0/10", "35.184.0.0/13", "130.211.0.0/16",
	}
	for _, cidr := range knownGcpCIDRs {
		_, cidrn, err := net.ParseCIDR(cidr)
		if err != nil {
			e("%+v", err)
			continue
		}
		if cidr != cidrn.String() {
			e("CIDR input doesn't match CIDR Network, %s vs %s", cidr, cidrn.String())
			continue
		}
		if cidrn.Contains(ip4) {
			//o("%s was matched to CIDR %s", ip, cidr)
			return true
		}
		//o("%s was not matched to CIDR %s", ip, cidr)
	} // end for cidr
	return false
} // end func CheckIpIsGcpLoadBalancer

func CheckIpIsIana(ip string) bool {
	// HC: 35.191.0.0/16, 130.211.0.0/22
	// GFE: 34.96.0.0/20, 34.127.192.0/18, 130.211.0.0/22, 35.191.0.0/16
	ip4 := net.ParseIP(ip)
	if ip4 == nil || ip4.IsPrivate() || ip4.IsUnspecified() || ip4.IsLoopback() {
		return false
	}
	ip = s("%s", ip4)
	knownIanaCIDRs := []string{
		"169.254.0.0/16",
	}
	for _, cidr := range knownIanaCIDRs {
		_, cidrn, err := net.ParseCIDR(cidr)
		if err != nil {
			e("%+v", err)
			continue
		}
		if cidr != cidrn.String() {
			e("CIDR input doesn't match CIDR Network, %s vs %s", cidr, cidrn.String())
			continue
		}
		if cidrn.Contains(ip4) {
			//o("%s was matched to CIDR %s", ip, cidr)
			return true
		}
		//o("%s was not matched to CIDR %s", ip, cidr)
	} // end for cidr
	return false
} // end func CheckIpIsIana

func GetAvailablePort(wantPort string, min, max int64) (havePort string, available bool, err error) {
	havePort = s("%s", wantPort)
	available, err = CheckIfTcpPortAvailable(havePort)
	if !available || err != nil {
		for iTestPort := min; iTestPort <= max; iTestPort++ {
			testPort := s("%d", iTestPort)
			if testPort != wantPort {
				available, err = CheckIfTcpPortAvailable(testPort)
				if available && err == nil {
					havePort = testPort
					return
				} // end if available
			} // end if testPort
		} // end for iTestPort
	} // end if available
	return
} // end func GetAvailablePort

// GetClientIP Gets the Remote User's Public IP Address, trying the proxy headers first before the standard remote address request variable
func GetClientIP(r *http.Request) string {
	var resolved_ip string = ""
	var forwarded_ip string = ""
	var forwarded_ips string = ""
	var isGcpLoadBalancer bool = false
	// Check if the request came through a proxy -- WARNING: This data can be spoofed and shouldn't be trusted for security uses
	forwarded_ips = r.Header.Get("X-Forwarded-For") // TODO TODO -- We should only use this if the resolved IP is a known load balancer IP
	if forwarded_ips == "" {
		forwarded_ips = r.Header.Get("x-forwarded-for")
	}
	//o("ns:forwarded_ips: %+v", forwarded_ips)
	if forwarded_ips == "" {
		rp := make(map[string]interface{})
		headers, err := AnyToHeadersMap(r.Header)
		if err != nil {
			e("No Headers Mapped (%T): %+v, Error: %+v", r.Header, r.Header, err)
			rp["headers"] = r.Header
		} else if len(headers) > 0 {
			rp["headers"] = headers
		} else {
			o("No Headers Mapped (%T): %+v", r.Header, r.Header)
			rp["headers"] = r.Header
		}
		//o("r[headers]: %+v", rp["headers"])
	}
	if forwarded_ips != "" {
		if strings.Contains(forwarded_ips, ",") {
			//o("ns:forwarded_ips: %+v", forwarded_ips)
			ips := StrSplit(forwarded_ips, ",")
			//o("ns:ips: %+v", ips)
			for _, ip := range ips {
				if ip != "" {
					//forwarded_ip = ip
					//o("ns:forwarded_ip: %+v", forwarded_ip)
					// X-Forwarded-For: <supplied-value>,<client-ip>,<load-balancer-ip>
					if CheckIpIsGcpLoadBalancer(ip) {
						isGcpLoadBalancer = true
						break
					} else {
						forwarded_ip = ip
						//o("ns:forwarded_ip: %+v", forwarded_ip)
					} // end if CheckIpIsGcpLoadBalancer
				} // end if ip
			} // end foreach ip
		} else {
			forwarded_ip = forwarded_ips
			//o("nc:forwarded_ip: %+v", forwarded_ip)
		} // end if forwarded_ips csv
	} // end if forwarded_ips
	//o("isGcpLoadBalancer: %+v, forwarded_ip: %+v", isGcpLoadBalancer, forwarded_ip)
	if !isGcpLoadBalancer || forwarded_ip == "" {
		remoteaddr_ip := r.RemoteAddr
		//o("ra:remoteaddr_ip: %+v", remoteaddr_ip)
		if remoteaddr_ip != "" {
			if strings.Contains(remoteaddr_ip, ":") {
				host_ip, _, err := net.SplitHostPort(remoteaddr_ip)
				//o("ra:host_ip: %+v", host_ip)
				if err == nil {
					parsed_ip := net.ParseIP(host_ip)
					//o("ra:parsed_ip: %+v", parsed_ip)
					if parsed_ip != nil {
						resolved_ip = s("%+v", parsed_ip)
					} // end if parsed_ip
				} // end if net.SplitHostPort
			} else {
				parsed_ip := net.ParseIP(remoteaddr_ip)
				//o("ra:parsed_ip: %+v", parsed_ip)
				if parsed_ip != nil {
					resolved_ip = s("%+v", parsed_ip)
				} // end if parsed_ip
			} // end if :
			if resolved_ip != "" {
				//o("ra:resolved_ip: %+v", resolved_ip)
				if CheckIpIsGcpLoadBalancer(resolved_ip) {
					isGcpLoadBalancer = true
				}
			}
		} // end if remoteaddr_ip
	} // end if forwarded_ip
	if isGcpLoadBalancer && forwarded_ip != "" {
		parsed_ip := net.ParseIP(forwarded_ip)
		//o("lb:parsed_ip: %+v", parsed_ip)
		if parsed_ip != nil {
			resolved_ip = s("%+v", parsed_ip)
			//o("lb:resolved_ip: %+v", resolved_ip)
		}
	} // end if detected_ip

	// Check the Remote Addr value of the HTTP Request
	if resolved_ip == "::1" {
		resolved_ip = "127.0.0.1"
	}
	o("%+v", resolved_ip)
	return resolved_ip
} // end func GetClientIP

func GetUserAgent(r *http.Request) string {
	userAgent := r.UserAgent()
	if userAgent == "" {
		userAgent = r.Header.Get("User-Agent")
	}
	return userAgent
} // end func GetUserAgent

func UserAgentIsCommonBrowser(userAgent string) bool {
	isCommonBrowser := false
	if userAgent != "" {
		for _, browser := range KnownCommonBrowsers {
			if StrContains(userAgent, browser, true) {
				isCommonBrowser = true
			} // end if StrContains
		} // end for range browser
	} // end if userAgent
	return isCommonBrowser
} // end func UserAgentIsCommonBrowser

func GetCpaasParamsFilter() []string {
	filter := []string{
		"AccountSid",
		"CallSid",
		"CallStatus",
		"Timestamp",
		"From",
		"To",
		"CallbackUrl",
		"CallUrl",
		"VoiceUrl",
		"StatusCallback",
		"StatusUrl",
		"Url",
		"Debug",
		"Async",
		"AsyncAmd",
		"MachineDetection",
		"AsyncAmdStatusCallback",
	}
	return filter
} // end func GetCpaasParamsFilter

func GetRequestCpaasFromUrl(r *http.Request, opt ...string) string {
	var requestCpaas string = ""
	if len(opt) > 0 {
		requestCpaas = opt[0]
	}
	pathStr := string(r.URL.Path)
	if StrContains(pathStr, "/avaya") {
		requestCpaas = "avaya"
	} else if StrContains(pathStr, "/emulator") {
		requestCpaas = "emulator"
	} else if StrContains(pathStr, "/telnyx") {
		requestCpaas = "telnyx"
	} else if StrContains(pathStr, "/twilio") || StrContains(pathStr, "/2010-04-01/accounts") {
		requestCpaas = "twilio"
	} else if requestCpaas != "" {
		e("Unable to determine the Request CPaaS Provider from the Path: %s, Using Default: %s", pathStr, requestCpaas)
	} else {
		e("Unable to determine the Request CPaaS Provider from the Path: %s, No Default Set", pathStr)
	}
	return requestCpaas
} // end func GetRequestCpaasFromUrl

func GetRequestProtocolPrefix(r *http.Request) string {
	// Check if the request came through a proxy -- WARNING: This data can be spoofed and shouldn't be trusted for security uses
	forwardedProto := r.Header.Get("X-Forwarded-Proto")
	if forwardedProto != "" {
		return forwardedProto + "://"
	}
	// Default to HTTPS unless its localhost
	protocolPrefix := "https://"
	hostStr := string(r.Host)
	if StrContains(hostStr, "localhost") == true || StrContains(hostStr, "127.0.0.1") == true || StrContains(hostStr, "::1") == true {
		protocolPrefix = "http://"
	}
	return protocolPrefix
} // end func GetRequestProtocolPrefix

func GetRequestHost(r *http.Request) string {
	// Check if the request came through a proxy -- WARNING: This data can be spoofed and shouldn't be trusted for security uses
	forwardedHost := r.Header.Get("X-Forwarded-Host")
	if forwardedHost != "" {
		return forwardedHost
	}
	hostStr := string(r.Host)
	return hostStr
} // end func GetRequestHost

func GetRequestBaseDomain(r *http.Request) string {
	// This will also have the domain:port if specified, lets remove the port to get the domain by itself
	protocolPrefix := GetRequestProtocolPrefix(r)
	hostStr := GetRequestHost(r)
	_, domain, _, _, err := ParseRawURL(protocolPrefix + hostStr)
	if err != nil {
		e("Failed to Parse Domain from URL %s%s", protocolPrefix, hostStr)
	}
	return domain
} // end func GetRequestBaseDomain

func GetRequestBaseUrl(r *http.Request, opt ...string) string {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	protocolPrefix := GetRequestProtocolPrefix(r)
	hostStr := GetRequestHost(r)
	requestBaseUrl := protocolPrefix + hostStr
	if len(opt) > 0 {
		opath := PathJoin(opt...)
		if opath != "" {
			requestBaseUrl = PathJoin(requestBaseUrl, opath)
		}
	}
	return requestBaseUrl
} // end func GetRequestBaseUrl

func GetRequestUri(r *http.Request, opt ...string) RequestUri {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var baseUrl string
	if len(opt) > 0 {
		baseUrl = GetRequestBaseUrl(r, opt...)
	} else {
		baseUrl = GetRequestBaseUrl(r, thisApiServicePrefix)
	}
	reqUri := ParseUri(baseUrl)
	reqUri.Method = r.Method
	return reqUri
} // end func GetRequestUri

func ParseUri(url string) RequestUri {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	scheme, domain, port, path, err := ParseRawURL(url)
	if err != nil {
		e("Failed to Parse Domain from URL %s", url)
	}

	var reqUri RequestUri
	reqUri.BaseUrl = url
	reqUri.Scheme = scheme
	reqUri.Domain = domain
	reqUri.Port = port
	reqUri.Path = path
	return reqUri
} // end func ParseUri

func GetRequestIdFromPath(requestPath string, exclude ...string) string {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var id string = ""

	// Remove /index.* from the path string
	indexPath := "/index\\.(html|php|asp|js|cgi|xhtml|htm|pl)" // Required index.* -- not indexRegex (optional)
	pattern, _ := regexp.Compile(indexPath)
	//o("requestPath: %s", requestPath)
	if pattern.MatchString(requestPath) {
		requestPath = pattern.ReplaceAllString(requestPath, "")
	}

	//o("requestPath: %s", requestPath)
	if SubStr(requestPath, -1) == "/" {
		requestPath = SubStr(requestPath, 0, -1)
	}
	//o("requestPath: %s", requestPath)
	requestPathLen := len(requestPath)
	if requestPathLen > thisApiServicePrefixLen {
		//o("RequestPath: %s, Checking Prefix: %s", requestPath, SubStr(requestPath, 0, thisApiServicePrefixLen))
		if SubStr(requestPath, 0, thisApiServicePrefixLen) == thisApiServicePrefix {
			requestPath = SubStr(requestPath, thisApiServicePrefixLen)
			requestPathLen = len(requestPath)
			//o("Adjusted RequestPath: %s", requestPath)
		}
	}
	if SubStr(requestPath, 0, 1) == "/" {
		requestPath = SubStr(requestPath, 1)
	}
	//o("requestPath: %s", requestPath)
	if StrContains(requestPath, "/") {
		s := StrSplit(requestPath, "/")
		//o("requestPath split: %+v", s)
		id = s[len(s)-1]
	} else {
		id = requestPath
	}
	//o("id: %s", id)
	if id != "" && len(exclude) > 0 {
		for _, ex := range exclude {
			if StrContains(ex, "/") {
				exp := StrSplit(ex, "/")
				if len(exp) > 0 {
					idx := GetRequestIdFromPath(requestPath, exp...)
					if idx == "" {
						id = ""
					}
				}
			} else if ex != "" && id == ex {
				id = ""
			}
		} // end foreach exclude
	} // end if id exclude
	//o("id: %s", id)
	return id
} // end func GetRequestIdFromPath

func MapFormParams(r *http.Request, inputKeys ...string) (map[string]interface{}, error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	params := make(map[string]interface{})
	getParams := make(map[string]interface{})
	multiPartParams := make(map[string]interface{})
	xWwwFormParams := make(map[string]interface{})
	jsonParams := make(map[string]interface{})
	lastErr := MakeNilError()
	filter := make(map[string]bool)
	anyKey := true
	if len(inputKeys) > 0 {
		for _, key := range inputKeys {
			if key != "" {
				anyKey = false
				filter[key] = true
			}
		}
	}

	// Always check the request's GET query parameters
	query := r.URL.Query()
	if len(query) > 0 {
		for key, value := range query {
			if key != "" {
				_, filterKey := filter[key]
				//o("anyKey: %+v filterKey: %+v", anyKey, filterKey)
				if anyKey || filterKey {
					for _, v := range value {
						v = UrlDecodeStr(v)
						_, exists := getParams[key]
						if exists == true {
							switch vv := getParams[key].(type) {
							case string:
								getParams[key] = []string{vv, v}
							case []string:
								getParams[key] = append(vv, v)
							default:
								e("getParams[%s] = %s   (type %T)", key, vv, vv)
							} // end switch type getParams[key]
						} else {
							getParams[key] = v
						} // end if exists
					} // end for v
				} // end if filterKey
			} // end if key
		} // end for range query
		//o("GET URL getParams: %+v", getParams)
		params = getParams
	} // end if query

	// Get the POST Parameters or JSON body depending on the Content-Type header
	post_form_data_found := false
	request_content_type := r.Header.Get("Content-Type")
	if request_content_type == "" {
		request_content_type = r.Header.Get("content-type") // try lowercase
	}
	//o("Request Content-Type: %s", request_content_type)
	if request_content_type != "" {
		var multipartReaderErr error
		if StrContains(request_content_type, "multipart/form-data") == true || StrContains(request_content_type, "multipart/mixed") == true {
			form, multipartReaderErr := r.MultipartReader()
			lastErr = multipartReaderErr
			//o("Multipart Reader Error: %+v", multipartReaderErr)
			if multipartReaderErr == nil {
				// Request is a Multipart Form [Content-Type: multipart/form-data]
			formParts:
				for {
					part, err := form.NextPart()
					if err == io.EOF {
						break formParts
					}
					key := part.FormName()
					if key != "" {
						post_form_data_found = true
						_, filterKey := filter[key]
						//o("anyKey: %+v filterKey: %+v", anyKey, filterKey)
						if anyKey || filterKey {
							buf := new(bytes.Buffer)
							buf.ReadFrom(part)
							value := strings.TrimSpace(s("%s", buf.String()))
							v, err := url.QueryUnescape(value)
							if err != nil {
								lastErr = err
							}
							_, exists := multiPartParams[key]
							if exists == true {
								switch vv := multiPartParams[key].(type) {
								case string:
									multiPartParams[key] = []string{vv, v}
								case []string:
									multiPartParams[key] = append(vv, v)
								default:
									e("multiPartParams[%s] = %s   (type %T)", key, vv, vv)
								} // end switch type multiPartParams[key]
							} else {
								multiPartParams[key] = v
							} // end if exists
						} // end if filterKey
					} // end if key
				} // end for
				//o("Multipart Params: %+v", multiPartParams)
				if len(multiPartParams) > 0 {
					params, _ = MergeMaps(multiPartParams, getParams)
				}
			} else {
				e("Multipart Reader Error: %+v", multipartReaderErr)
			} // end if multipartReaderErr
		} // end if content-type multipart
		if StrContains(request_content_type, "application/x-www-form-urlencoded") == true || multipartReaderErr != nil || len(multiPartParams) == 0 {
			//o("Count of multiPartParams %d", len(multiPartParams))
			if multipartReaderErr != nil || len(multiPartParams) == 0 {
				// Request is 'normal' POST form data [Content-Type: application/x-www-form-urlencoded]
				r.ParseForm()

				//o("POST Form: %s", r.Form)
				if len(r.Form) > 0 {
					lastErr = nil
					for key, value := range r.Form {
						if key != "" {
							post_form_data_found = true
							_, filterKey := filter[key]
							//o("anyKey: %+v filterKey: %+v", anyKey, filterKey)
							if anyKey || filterKey {
								for _, v := range value {
									v = UrlDecodeStr(v)
									_, exists := xWwwFormParams[key]
									if exists == true {
										switch vv := xWwwFormParams[key].(type) {
										case string:
											xWwwFormParams[key] = []string{vv, v}
										case []string:
											xWwwFormParams[key] = append(vv, v)
										default:
											e("xWwwFormParams[%s] = %s   (type %T)", key, vv, vv)
										} // end switch type xWwwFormParams[key]
									} else {
										xWwwFormParams[key] = v
									} // end if exists
								} // end for v
							} // end if filterKey
						} // end if key
					} // end for range query
				} // end if r.Form
				//o("x-www-form Params: %+v", xWwwFormParams)
				if len(xWwwFormParams) > 0 {
					params, _ = MergeMaps(params, xWwwFormParams, getParams)
				}
			} // end if multipartReaderErr
		} // end if request_content_type x-www-form
	} // end if request_content_type (empty)
	//o("POST Form Fields Found %+v, Count of Params %d", post_form_data_found, (len(multiPartParams) + len(xWwwFormParams)))
	if post_form_data_found == false || (len(multiPartParams)+len(xWwwFormParams)) == 0 {
		// Otherwise try JSON Body (Content-Type: application/json )
		err := GetRequestBodyJSON(r, &jsonParams)
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
		}
		//o("JSON Body params: %+v", jsonParams)
		if len(jsonParams) > 0 {
			params, _ = MergeMaps(params, jsonParams, getParams)
		}
	} // end if post_form_data_found
	//o("All Request Params: %+v", params)
	return params, lastErr
} // end func MapFormParams

func CopyMap(original map[string]interface{}) (copy map[string]interface{}, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// Any other method just creates pointers to the same underlying map... JSON encode/decode breaks the map pointer
	copy = make(map[string]interface{})
	copyBytes, err := JsonEncode(original)
	if err != nil {
		return
	}
	err = JsonDecode(copyBytes, &copy)
	return
} // end func CopyMap

func CopySlice(original []interface{}) (copy []interface{}, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// Any other method just creates pointers to the same underlying map... JSON encode/decode breaks the map pointer
	copy = make([]interface{}, 0)
	copyBytes, err := JsonEncode(original)
	if err != nil {
		return
	}
	err = JsonDecode(copyBytes, &copy)
	return
} // end func CopySlice

func MergeAny(existing interface{}, inputs ...interface{}) (result interface{}, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	result = existing
	for _, input := range inputs {
		result, err = MergeValues(result, input)
		if err != nil {
			return
		}
	}
	return
} // end func MergeAny

func MergeMaps(existing map[string]interface{}, inputs ...interface{}) (map[string]interface{}, error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	for _, input := range inputs {
		if err := MergeIntoMap(existing, input); err != nil {
			return nil, err
		}
	}
	return existing, nil
} // end func MergeMaps

func MergeIntoMap(dest map[string]interface{}, incoming interface{}) error {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	incomingVal := reflect.ValueOf(incoming)

	switch incomingVal.Kind() {
	case reflect.Map:
		for _, key := range incomingVal.MapKeys() {
			keyStr := s("%+v", key.Interface())
			val := incomingVal.MapIndex(key).Interface()
			if existingVal, exists := dest[keyStr]; exists {
				mergedVal, err := MergeValues(existingVal, val)
				if err != nil {
					return err
				}
				dest[keyStr] = mergedVal
			} else {
				dest[keyStr] = val
			}
		}
	case reflect.Struct:
		for i := 0; i < incomingVal.NumField(); i++ {
			field := incomingVal.Type().Field(i)
			keyStr := field.Name
			val := incomingVal.Field(i).Interface()
			if existingVal, exists := dest[keyStr]; exists {
				mergedVal, err := MergeValues(existingVal, val)
				if err != nil {
					return err
				}
				dest[keyStr] = mergedVal
			} else {
				dest[keyStr] = val
			}
		}
	case reflect.Slice, reflect.Array:
		sliceLen := 0
		if incomingVal.Type().Kind() == reflect.Slice && incomingVal.Len() > 0 {
			sliceLen = incomingVal.Len()
		} else if incomingVal.Type().Kind() == reflect.Array && incomingVal.Type().Len() > 0 {
			sliceLen = incomingVal.Type().Len()
		}
		if sliceLen > 0 {
			for keyi := 0; keyi < sliceLen; keyi++ {
				// Stringify the key
				ki := s("%d", keyi)

				// Interfaceify the value
				vi := incomingVal.Index(keyi).Interface()

				dest[ki] = vi
			} // end for range input
		} // end if sliceLen
	default:
		return eer("Unsupported Type %s: %+v", incomingVal.Kind(), incomingVal.Interface())
	}

	return nil
} // end func MergeIntoMap

func MergeSlices(existing []interface{}, incoming interface{}) (result []interface{}, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	result = existing
	incomingVal := reflect.ValueOf(incoming)
	switch incomingVal.Kind() {
	case reflect.Slice, reflect.Array:
		sliceLen := 0
		if incomingVal.Type().Kind() == reflect.Slice && incomingVal.Len() > 0 {
			sliceLen = incomingVal.Len()
		} else if incomingVal.Type().Kind() == reflect.Array && incomingVal.Type().Len() > 0 {
			sliceLen = incomingVal.Type().Len()
		}
		if sliceLen > 0 {
			for i := 0; i < sliceLen; i++ {
				if i < len(result) {
					result[i], err = MergeValues(result[i], incomingVal.Index(i).Interface())
					if err != nil {
						return
					}
				} else {
					result = append(result, incomingVal.Index(i).Interface())
				}
			} // end foreach slice index
		} // end if incomingVal.Len
	default:
		e("Incoming Value is not a Slice or Array! %T: %+v", incoming, incoming)
	} // end switch incomingVal.Kind

	return
} // end func MergeSlices

func MergeValues(existing, incoming interface{}) (interface{}, error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	existingVal := reflect.ValueOf(existing)
	incomingVal := reflect.ValueOf(incoming)
	var err error

	switch existingVal.Kind() {
	case reflect.Slice, reflect.Array:
		switch incomingVal.Kind() {
		case reflect.Slice, reflect.Array:
			dest := make([]interface{}, 0)
			dest, err = MergeSlices(dest, existingVal.Interface())
			if err != nil {
				return nil, err
			}
			dest, err = MergeSlices(dest, incomingVal.Interface())
			if err != nil {
				return nil, err
			}
			return dest, nil
		default:
			return incoming, nil
		} // end switch incomingVal.Kind
	case reflect.Map, reflect.Struct:
		switch incomingVal.Kind() {
		case reflect.Map, reflect.Struct:
			dest := make(map[string]interface{})
			if err = MergeIntoMap(dest, existingVal.Interface()); err != nil {
				return nil, err
			}
			if err = MergeIntoMap(dest, incomingVal.Interface()); err != nil {
				return nil, err
			}
			return dest, nil
		default:
			return incoming, nil
		} // end switch incomingVal.Kind
	default:
		// For other types, incoming overrides existing
		return incoming, nil
	} // end switch existingVal.Kind
} // end func MergeValues

// Execute New Outgoing REST Request
func NewRestApiRequest[T any](method string, url string, response *T, opt ...interface{}) (rp RestApiResponse, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	// Optional Params, Defaults
	var rx RestApiRequest
	var default_timeout int64 = 10
	var default_caching int64 = 0
	rx_start := TimeNow()
	rx.Method = method
	rx.URL = url
	rx.URI = ParseUri(url)
	rx.Timeout = 0
	rx.Caching = 0
	rx.Headers = make(map[string][]string)
	BodyReader := MakeNilReader()
	if len(opt) > 0 {
		// Optional:
		//      rx/Body (string OR []byte OR io.Reader)
		//      rx/Params (map[string]interface{})
		//      rx/Headers (map[string]string)
		//      rx/Timeout (time.Duration OR int64) [1st int input]
		//      rx/Caching (time.Duration OR int64) [2nd int input]
		for _, opt := range opt {
			switch input := opt.(type) {
			case int:
				if rx.Timeout == 0 {
					rx.Timeout = int64(input)
				} else if rx.Caching == 0 {
					rx.Caching = int64(input)
				}
			case int32:
				if rx.Timeout == 0 {
					rx.Timeout = int64(input)
				} else if rx.Caching == 0 {
					rx.Caching = int64(input)
				}
			case int64:
				if rx.Timeout == 0 {
					rx.Timeout = input
				} else if rx.Caching == 0 {
					rx.Caching = input
				}
			case time.Duration:
				rx.Timeout = ParseDuration(input)

			case http.Header: // underlying is map[string][]string
				rx.Headers, err = AnyToHeadersMap(input)
			case map[string]string:
				rx.Headers, err = AnyToHeadersMap(input)
			case map[string][]string:
				rx.Headers, err = AnyToHeadersMap(input)
			case map[string]interface{}:
				rx.Params = input
			case []byte:
				rx.Body = input
				BodyReader = bytes.NewReader(input)
			case string:
				rx.Body = []byte(input)
				BodyReader = strings.NewReader(input)
			case io.Reader:
				BodyReader = input
			default:
				t := reflect.TypeOf(input)
				if t.Kind() == reflect.Struct {
					rx.Params = StructToMapStringInterface(input)
				}
			} // end switch input
		} // end for opt
	} // end if opt

	if rx.Timeout == 0 {
		rx.Timeout = default_timeout
	}
	if rx.Caching == 0 {
		rx.Caching = default_caching
	}

	var myTransport http.RoundTripper = &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: TimeDuration(rx.Timeout),
	}

	rp.Request = rx
	rp.SetCode(http.StatusRequestTimeout)
	var req *http.Request
	err = nil

	if rx.Caching > 0 {
		isJsonEncoded := false
		isResponseLoaded := false
		found, crp, err := GetCachedRestApiRequest(rx, response)
		if found {
			o("Cached Response: Found Response Object crp (%T): %+v", crp, crp)
			if err != nil {
				o("Cached Response with Errors: %+v", err)
				o("Cached Response with Errors: crp.Body (%T): %+v", crp.Body, crp.Body)
				o("Cached Response with Errors: response (%T): %+v", *response, *response)
			} else {
				o("Cached Response with No Errors: crp.Body (%T): %+v", crp.Body, crp.Body)
				o("Cached Response with No Errors: response (%T): %+v", *response, *response)

				// Check the Response Content-Type
				crp_content_types, ctOk := crp.Headers["Content-Type"]
				//o("Cached Response crp.Headers[Content-Type] (%+v): %+v", ctOk, crp_content_types)
				if ctOk && len(crp_content_types) > 0 {
					crp_content_type := crp_content_types[0]
					//o("Cached Response crp.Headers[Content-Type][0]: %+v", crp_content_type)
					if StrContains(crp_content_type, "application/json") == true {
						isJsonEncoded = true
					} // end if json
				} // end if Content-Type
				if !isJsonEncoded {
					crp_content_types, ctOk := crp.Request.Headers["Accept"]
					//o("Cached Response crp.Request.Headers[Accept] (%+v): %+v", ctOk, crp_content_types)
					if ctOk && len(crp_content_types) > 0 {
						crp_content_type := crp_content_types[0]
						//o("Cached Response crp.Request.Headers[Accept][0]: %+v", crp_content_type)
						if StrContains(crp_content_type, "application/json") == true {
							isJsonEncoded = true
						} // end if json
					} // end if Content-Type
				} // end if isJsonEncoded
			} // end if err
			if isJsonEncoded && err == nil {
				crBodyBytes := AnyToByte(crp.Body)
				//o("Cached Response JSON Encoded: crp.Body Bytes: %+v", string(crBodyBytes[:]))
				if len(crBodyBytes) > 0 {
					err = JsonDecode(crBodyBytes, response)
					if err != nil {
						e("Cached Response: Failed to JsonDecode the Response: %+v", err)
						err = nil
					} else {
						isResponseLoaded = true
					} // end if err
				} // end if crBodyBytes
			} // end if isJsonEncoded
			//o("Cached Response isResponseLoaded (%+v) and isJsonEncoded (%+v)", isResponseLoaded, isJsonEncoded)
			if !isResponseLoaded && !isJsonEncoded {
				var tOk bool = false
				responseBodyT := *response
				responseBodyIntf := interface{}(crp.Body)
				responseBodyT, tOk = responseBodyIntf.(T)
				if tOk {
					response = &responseBodyT
					isResponseLoaded = true
					err = nil
				}
			} // end if isJsonEncoded
			//o("Cached Response found (%+v), isResponseLoaded (%+v), err (%+v)", found, isResponseLoaded, err)
			if isResponseLoaded && err == nil {
				o("Cached Response: Got the Response Mapped Response Body: %+v", *response)
				return crp, err
			} // end if isResponseLoaded
		} // end if found
	} // end if rx.Caching

	// Build the Request
	switch rx.Method {
	case http.MethodPost, http.MethodPut:
		if rx.Body == nil {
			dataByte, err := JsonEncode(rx.Params)
			if err != nil {
				o("JSON marshal Failed, Error: %+v", err)
			}
			//o("%s Request Body: %s", rx.Method, string(dataByte[:]))
			rx.Body = dataByte
			BodyReader = bytes.NewBuffer(dataByte)
		}
		req, err = http.NewRequest(rx.Method, rx.URL, BodyReader)
	case http.MethodGet, http.MethodDelete:
		req, err = http.NewRequest(rx.Method, rx.URL, nil)
	default:
		err := er("%s Request Method Not Implemented in Function Yet", rx.Method)
		e("%+v", err)
		rp.SetCode(400)
		return rp, err
	} // end switch rx.Method
	if err != nil {
		err := er("%s Request to %s got the error: %+v", rx.Method, rx.URL, err)
		e("%+v", err)
		if rp.GetCode() == 0 {
			rp.SetCode(500)
		}
		return rp, err
	}

	// Build the Headers
	contentTypeHeaderSet := false
	acceptHeaderSet := false
	if len(rx.Headers) > 0 {
		for key, vals := range rx.Headers {
			if key == "Accept" {
				acceptHeaderSet = true
			} else if key == "Content-Type" {
				contentTypeHeaderSet = true
			}
			for hi := 0; hi < len(vals); hi++ {
				if hi == 0 {
					req.Header.Set(key, vals[hi])
				} else {
					req.Header.Add(key, vals[hi])
				}
			} // end foreach val
		} // end foreach header
	} // end if headers
	if contentTypeHeaderSet == false {
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	}
	if acceptHeaderSet == false {
		req.Header.Set("Accept", "application/json")
	}

	// Init the HTTP Client
	client := &http.Client{
		Transport: myTransport,
	}

	// Execute the Request
	o("[METHOD: %s] [URL: %s] [HEADERS: %+v] [DATA: %+v] [TIMEOUT: %d]", rx.Method, rx.URL, rx.Headers, rx.Params, rx.Timeout)
	r, err := client.Do(req)
	if err != nil {
		e("Outgoing Request Failed %s %s: %+v", rx.Method, rx.URL, err)
	} else if r != nil {
		if r.Body != nil {
			rp.SetCode(r.StatusCode)
			bodyByte, err := GetResponseBodyBytes(r)
			if err != nil {
				e("Failed to Read the response body for %s %s: %+v", rx.Method, rx.URL, err)
			}
			if len(bodyByte) > 0 {
				r_content_type := r.Header.Get("Content-Type")
				if StrContains(r_content_type, "application/json") == true {
					err := JsonDecode(bodyByte, response)
					if err != nil {
						e("JSON Unmarshal Failed with Error %+v for response body: %s", err, string(bodyByte[:]))
						rp.Body = string(bodyByte[:])
					} else {
						rp.Body = *response
					} // end if json.Unmarshal
				} else {
					rp.Body = string(bodyByte[:])
				} // end if Content-Type
			} // end if len bodyByte
		} // end if r.Body
	} // end if r

	rp.SetRuntime(Runtime(rx_start))
	if err == nil && rp.GetCode() == 200 && rx.Caching > 0 {
		saved, serr := SaveCachedRestApiRequest(rx, rp)
		if saved == false || serr != nil {
			e("Failed to Save the Rest Api Request Response to Cache: (%+v) %+v", saved, serr)
		}
	}
	//o("Got Real Response: %+v", rp)
	return rp, err
} // end func NewRestApiRequest

func GetCachedRestApiRequest[T any](rx RestApiRequest, response *T) (found bool, rp RestApiResponse, err error) {
	found = false
	err = nil
	if rx.Caching <= 0 {
		return
	}

	// Hash the request object
	hashed, err := GetRestApiRequestHashStr(rx)
	if hashed == "" || err != nil {
		return
	}

	// Set the expected save Path
	cachedRequestPath := PathJoin(thisApiServicePrefix, "files", thisGcsBaseSavePath, "json", "rx", "cache", hashed+".json")

	// Check the Storage Bucket
	found, cachedRequestPath, size, updated, _, err := GcsReadFileJSON(thisGcsStorageBucket, cachedRequestPath, &rp)
	if !found || size == 0 || err != nil || TimeSince(updated) > TimeDuration(rx.Caching) {
		// GCS File doesn't exist or is not readable or is older than caching config
		if found && err != nil {
			e("Cached Response Failed to Read the File from GCS %s: Error: %+v", cachedRequestPath, err)
		}
		found = false
		return
	} else {
		o("Cached Response found and loaded from %s (%+v, %d, %+v) (%T): %+v", cachedRequestPath, TimeSince(updated), size, found, rp, rp)
		found = true
	}

	rpRxCts, ok := rp.Request.Headers["Content-Type"]
	if ok && len(rpRxCts) > 0 {
		rpRxCt := rpRxCts[0]
		if rpRxCt != "" && StrContains(rpRxCt, "json") && !IsEmpty(rp.Request.Body) {
			o("rp.Request.Body: (%T): %+v", rp.Request.Body, rp.Request.Body)
			rxBody := make(map[string]interface{})
			var rxdecoded bool = false
			var rxerr error = nil
			switch rpRxBody := rp.Request.Body.(type) {
			case []byte:
				rxerr = JsonDecode(rpRxBody, &rxBody)
				rxdecoded = true
			case string:
				rxerr = JsonDecode(rpRxBody, &rxBody)
				rxdecoded = true
			} // end switch rp.Request.Body.(type)
			if rxdecoded && rxerr != nil {
				e("rp.Request.Body: Failed to Decode the Request Body from: (%T) %+v to (%T): Error: %+v", rp.Request.Body, rp.Request.Body, rxBody, rxerr)
				//err = rxerr
			} else if rxdecoded {
				o("rp.Request.Body: Successfully Decoded the Request Body from: (%T) %+v to (%T) %+v", rp.Request.Body, rp.Request.Body, rxBody, rxBody)
				rp.Request.Body = rxBody
			} // end if rxdecoded
		} // end if rpRxCt
	} // end if rpRxCts

	setAgain := true
	isJsonEncoded := false
	rpCts, ok := rp.Headers["Content-Type"]
	o("rp.Headers[Content-Type] (%+v): %+v", ok, rpCts)
	if ok && len(rpCts) > 0 {
		rpCt := rpCts[0]
		o("rp.Headers[Content-Type][0]: %+v", rpCt)
		if rpCt != "" && StrContains(rpCt, "json") {
			isJsonEncoded = true
		}
	}
	rpxCts, ok := rp.Request.Headers["Accept"]
	o("rp.Request.Headers[Accept] (%+v): %+v", ok, rpxCts)
	if ok && len(rpxCts) > 0 {
		rpxCt := rpxCts[0]
		o("rp.Request.Headers[Accept][0]: %+v", rpxCt)
		if rpxCt != "" && StrContains(rpxCt, "json") {
			isJsonEncoded = true
		}
	}
	if isJsonEncoded {
		o("rp.Body: (%T): %+v", rp.Body, rp.Body)
		var rpdecoded bool = false
		var rpencoded bool = false
		var replaceBody bool = false
		var jrperr error = nil
		var rperr error = nil
		jsonRpBody := make([]byte, 0)
		switch rpBody := rp.Body.(type) {
		case map[string]interface{}:
			jsonRpBody, jrperr = JsonEncode(rpBody)
			rpencoded = true
			if jrperr == nil {
				rperr = JsonDecode(jsonRpBody, response)
				rpdecoded = true
			}
		case []interface{}:
			jsonRpBody, jrperr = JsonEncode(rpBody)
			rpencoded = true
			if jrperr == nil {
				rperr = JsonDecode(jsonRpBody, response)
				rpdecoded = true
			}
		case []byte:
			rperr = JsonDecode(rpBody, response)
			rpdecoded = true
			replaceBody = true
		case string:
			rperr = JsonDecode(rpBody, response)
			rpdecoded = true
			replaceBody = true
		} // end switch rp.Body.(type)
		if rpencoded && jrperr != nil {
			e("rp.Body: Failed to Encode the Response Body from: (%T) %+v: Error: %+v", rp.Body, rp.Body, jrperr)
			err = jrperr
		} else if rpdecoded && rperr != nil {
			e("rp.Body: Failed to Decode the Response Body from: (%T) %+v to (%T) %+v: Error: %+v", rp.Body, rp.Body, *response, *response, rperr)
			err = rperr
		} else if rpdecoded {
			o("rp.Body: Successfully Decoded the Response Body from: (%T) %+v to (%T) %+v", rp.Body, rp.Body, *response, *response)
			setAgain = false
			if replaceBody {
				rp.Body = *response
				err = nil
				return
			}
		} // end if rpdecoded
	} // end if isJsonEncoded
	isEmptyResponse := IsEmpty(response)
	isEmptyBody := IsEmpty(rp.Body)
	o("isJsonEncoded: %+v rp.Headers: %+v rp.Request.Headers: %+v", isJsonEncoded, rp.Headers, rp.Request.Headers)
	o("isEmptyResponse: %+v response (%T): %+v", isEmptyResponse, *response, *response)
	o("isEmptyBody: %+v rp.Body (%T): %+v", isEmptyBody, rp.Body, rp.Body)
	if isJsonEncoded && isEmptyResponse && !isEmptyBody {
		setAgain = true
		rpBody := make([]byte, 0)
		var err error = nil
		switch rpbt := rp.Body.(type) {
		case []byte:
			rpBody = rpbt
		case map[string]interface{}:
			rpBody, err = JsonEncode(rpbt)
		case []interface{}:
			rpBody, err = JsonEncode(rpbt)
		default:
			rpBody = AnyToByte(rpbt)
			err = er("Body is another type (%T): %+v", rpbt, rpbt)
		} // end switch rpbt
		if err != nil {
			e("Failed to JsonEncode, Error: %+v -- rpBody: %+v", err, rp.Body)
		} else {
			err = JsonDecode(rpBody, response)
			if err != nil {
				e("Failed to JsonDecode, Error: %+v -- rpBody: %s", err, string(rpBody[:]))
			} else {
				o("JsonDecoded: rp.Body (%T): %+v", rp.Body, rp.Body)
				o("JsonDecoded: Response (%T): %+v", *response, *response)
				setAgain = false
			} // end if err JsonDecode response
		} // end if err JsonEncode rp.Body
	} // end if IsEmpty response
	o("setAgain: %+v", setAgain)
	if setAgain {
		// Try to coerce the body into the response pointer
		rpBodyT := rp.Body.(T)
		response = &rpBodyT
		o("setAgain: rp.Body (%T): %+v", rp.Body, rp.Body)
		o("setAgain: rpBodyT (%T): %+v", rpBodyT, rpBodyT)
		o("setAgain: Response (%T): %+v", *response, *response)
	}
	return
} // end func GetCachedRestApiRequest

func SaveCachedRestApiRequest(rx RestApiRequest, rp RestApiResponse) (saved bool, err error) {
	saved = false
	err = nil
	if rp.GetCode() != 200 || rx.Caching <= 0 {
		return
	}

	hashed, err := GetRestApiRequestHashStr(rx)
	if hashed == "" || err != nil {
		return
	}

	rpRxCts, ok := rp.Request.Headers["Content-Type"]
	if ok && len(rpRxCts) > 0 {
		rpRxCt := rpRxCts[0]
		if rpRxCt != "" && StrContains(rpRxCt, "json") && !IsEmpty(rp.Request.Body) {
			switch rpRxBody := rp.Request.Body.(type) {
			case []byte:
				mapRpRxBody := make(map[string]interface{})
				jrperr := JsonDecode(rpRxBody, &mapRpRxBody)
				if jrperr == nil {
					rp.Request.Body = interface{}(mapRpRxBody)
				}
			case string:
				mapRpRxBody := make(map[string]interface{})
				jrperr := JsonDecode(rpRxBody, &mapRpRxBody)
				if jrperr == nil {
					rp.Request.Body = interface{}(mapRpRxBody)
				}
			} // end switch rpRxBody
		} // end if json
	} // end if rp.Request.Headers

	requestIsJson := false
	rpCts, ok := rp.Headers["Content-Type"]
	//o("rp.Headers[Content-Type] (%+v): %+v", ok, rpCts)
	if ok && len(rpCts) > 0 {
		rpCt := rpCts[0]
		o("rp.Headers[Content-Type][0]: %+v", ok, rpCt)
		if rpCt != "" && StrContains(rpCt, "json") && !IsEmpty(rp.Request.Body) {
			requestIsJson = true
		}
	}
	if !requestIsJson {
		rpCts, ok := rp.Request.Headers["Accept"]
		//o("rp.Request.Headers[Accept] (%+v): %+v", ok, rpCts)
		if ok && len(rpCts) > 0 {
			rpCt := rpCts[0]
			o("rp.Request.Headers[Accept][0]: %+v", ok, rpCt)
			if rpCt != "" && StrContains(rpCt, "json") && !IsEmpty(rp.Request.Body) {
				requestIsJson = true
			}
		}
	}
	if requestIsJson {
		switch rpBody := rp.Body.(type) {
		case []byte:
			mapRpBody := make(map[string]interface{})
			jrperr := JsonDecode(rpBody, &mapRpBody)
			if jrperr == nil {
				rp.Body = interface{}(mapRpBody)
			}
		case string:
			mapRpBody := make(map[string]interface{})
			jrperr := JsonDecode(rpBody, &mapRpBody)
			if jrperr == nil {
				rp.Body = interface{}(mapRpBody)
			}
		} // end switch rpRxBody
	} // end if requestIsJson

	// JSON encode the request
	rpJsonBytes, err := JsonEncode(rp)
	if err != nil {
		e("Failed to JSON encode the RestApiRequest Response to be cached: %+v", err)
		return
	}

	// Set the Storage Path
	cachedRequestPath := PathJoin(thisApiServicePrefix, "files", thisGcsBaseSavePath, "json", "rx", "cache", hashed+".json")

	// Set the metadata
	contentType := "application/json"
	contentDisposition := "inline; filename=" + PathBasename(cachedRequestPath)
	metadata := make(map[string]string)
	metadata["language"] = "en-US"

	// Save the Defaults to GCS
	size, err := GcsSaveFile(thisGcsStorageBucket, cachedRequestPath, rpJsonBytes, contentType, contentDisposition, metadata)
	if size == 0 || err != nil {
		e("Failed to Save the Cached RestApiRequest Response: %+v: %+v", err, string(rpJsonBytes[:]))
		saved = false
	} else {
		o("Successfully Saved the Cached RestApiRequest Response: %+v", string(rpJsonBytes[:]))
		saved = true
	}

	return
} // end func SaveCachedRestApiRequest

func GetRestApiRequestHashStr(rx RestApiRequest) (hashed string, err error) {
	hashed = ""
	rxb, err := JsonEncode(rx)
	if err != nil {
		return
	}
	hashed = HashStr(rxb)
	return
} // end func GetRestApiRequestHashStr

// GetResponseBodyBytes is used for reading the responses of outgoing requests from our service
func GetResponseBodyBytes(r *http.Response) (bodyByte []byte, err error) {
	defer r.Body.Close()
	bodyByte, err = ioutil.ReadAll(r.Body)
	if err != nil {
		e("Failed to Read the Response JSON Body: %+v", err)
		return
	}

	// Check if the server actually replied with compressed data
	if BytesGzipped(bodyByte) {
		// Gzip Decode the bytes here
		var reader io.ReadCloser
		buf := bytes.NewBuffer(bodyByte)
		reader, err = gzip.NewReader(buf)
		if err != nil {
			e("Failed to initialize the gzip Reader: %+v", err)
			return
		}
		gzBytes := make([]byte, 0)
		defer reader.Close()
		gzBytes, err = ioutil.ReadAll(reader)
		if err != nil {
			e("Failed to Read the gzip Response Body: %+v", err)
			return
		}
		bodyByte = gzBytes
	}
	//o("%s", string(bodyByte[:]))
	err = nil
	return
} // end func GetResponseBodyBytes

// GetResponseBodyJSON is used for reading the responses of outgoing requests from our service
func GetResponseBodyJSON[T any](r *http.Response, jsonObj *T) error {
	bodyByte, err := GetResponseBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bodyByte) > 0 {
		err = JsonDecode(bodyByte, &jsonObj)
		if err != nil {
			e("Failed to Unmarshal the Response JSON Body: %+v", err)
			e("Body Bytes: %s", string(bodyByte[:]))
			return err
		}
	} // end if bodyByte
	return nil
} // end func GetResponseBodyJSON

// GetRequestBodyBytes is used for reading the request body of incoming requests to our service
func GetRequestBodyBytes(r *http.Request) (bodyByte []byte, err error) {
	defer r.Body.Close()
	bodyByte, err = ioutil.ReadAll(r.Body)
	if err != nil {
		e("Failed to Read the Request JSON Body: %+v", err)
		return
	}

	// Check if the client actually sent compressed data
	if BytesGzipped(bodyByte) {
		// Gzip Decode the bytes here
		var reader io.ReadCloser
		buf := bytes.NewBuffer(bodyByte)
		reader, err = gzip.NewReader(buf)
		if err != nil {
			e("Failed to initialize the gzip Reader: %+v", err)
			return
		}
		gzBytes := make([]byte, 0)
		defer reader.Close()
		gzBytes, err = ioutil.ReadAll(reader)
		if err != nil {
			e("Failed to Read the gzip Request Body: %+v", err)
			return
		}
		bodyByte = gzBytes
	}
	//o("%s", string(bodyByte[:]))
	err = nil
	return
} // end func GetRequestBodyBytes

// GetRequestBodyJSON is used for reading the request body of incoming requests to our service
func GetRequestBodyJSON[T any](r *http.Request, jsonObj *T) error {
	bodyByte, err := GetRequestBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bodyByte) > 0 {
		err = JsonDecode(bodyByte, &jsonObj)
		if err != nil {
			e("Failed to Unmarshal the Request JSON Body: %+v", err)
			e("Body Bytes: %s", string(bodyByte[:]))
			return err
		}
	} // end if bodyByte
	return nil
} // end func GetRequestBodyJSON

func RecoverErrorStack(files ...string) {
	if errr, ok := recover().(error); errr != nil && ok {
		files = append(files, "main.go", "shared.go")
		t := Trace(3, files...)
		e("[%s:%s():%d]  Recover: %+v", t.File, t.Func, t.Line, errr)
		if ok, _ := InStringSlice(t.File, files); !ok {
			tb := Traceback(3, 25, files...)
			e("[%s:%s():%d]  Traceback: %+v", t.File, t.Func, t.Line, tb)
			e("[%s:%s():%d]  Stack: %s", t.File, t.Func, t.Line, string(debug.Stack()))
		}
	} // end if errr
} // end func RecoverErrorStack

func RecoverErrorStackRequest(w http.ResponseWriter, r *http.Request, files ...string) {
	if errr, ok := recover().(error); errr != nil && ok {
		files = append(files, "main.go", "shared.go")
		t := Trace(3, files...)
		e("[%s:%s():%d]  Recover: %+v", t.File, t.Func, t.Line, errr)
		if ok, _ := InStringSlice(t.File, files); !ok {
			tb := Traceback(3, 25, files...)
			e("[%s:%s():%d]  Traceback: %+v", t.File, t.Func, t.Line, tb)
			e("[%s:%s():%d]  Stack: %s", t.File, t.Func, t.Line, string(debug.Stack()))
		}
		InternalServerErrorHandler(w, r, s("%+v", errr))
	} // end if errr
} // end func RecoverErrorStackRequest

func TraceFile() string {
	t := Trace(1)
	return t.File
} // end func TraceFile

func TraceLine() int {
	t := Trace(1)
	return t.Line
} // end func TraceLine

func TraceFunc() string {
	t := Trace(1)
	fn := t.Func
	if StrContains(fn, ".func") {
		fns := StrSplit(fn, ".")
		fn = fns[0]
	}
	return fn
} // end func TraceFunc

func Trace(start int, files ...string) TraceType {
	tb := Traceback(start+1, 1, files...)
	t := tb.Stack[0]
	return t
} // end func Trace

func Traceback(start int, limit int, files ...string) TraceSlice {
	var t TraceSlice
	tbFuncs := []string{
		"Traceback", "Trace", "TraceFile", "TraceLine", "TraceFunc",
		"RecoverErrorStack", "RecoverErrorStackRequest",
		"LogFunc", "LogMsg", "s", "o", "oom", "d", "e", "er", "eer", "erm", "p",
	}
	t.Stack = make([]TraceType, 0)
	if start < 1 {
		start = 1 // 0 is runtime.Caller
	}
	for i := 1; i < limit+20; i++ {
		if i >= start {
			var l TraceType
			pc, file, line, ok := runtime.Caller(i)
			if ok {
				l.Frame = i
				l.File = PathBasename(file)
				l.Line = line
				l.Func = "?"
				fn := runtime.FuncForPC(pc)
				if fn != nil {
					fnn := fn.Name()
					if fnn != "" {
						l.Func = fnn
					}
				}
				if SubStr(l.Func, 0, 5) == "main." {
					l.Func = SubStr(l.Func, 5)
				}
				if SubStr(l.Func, -5) == "[...]" {
					l.Func = SubStr(l.Func, 0, -5)
				}
				logFunc, _ := InStringSlice(l.Func, tbFuncs)
				if !logFunc {
					ok = true
					if len(files) > 0 {
						ok, _ = InStringSlice(l.File, files)
					}
					if ok {
						t.Stack = append(t.Stack, l)
						if len(t.Stack) == limit {
							return t
						}
					}
				} // end if logFunc
			} else {
				i = limit + 99
			}
		} // end if i
	} // end for i
	return t
} // end func Trace

// RespondJSON is used for writing a JSON to the response body of incoming requests to our service
func RespondJSON(w http.ResponseWriter, r *http.Request, statusCode int, jsonObj any) {
	ise := http.StatusInternalServerError
	jsonByte, err := JsonEncode(jsonObj)
	if err != nil {
		e("RespondJSON: Error happened in JSON marshal. Error: %+v", err)
		if statusCode != ise {
			InternalServerErrorHandler(w, r, err.Error())
		} else {
			errorBody := s("%s: %+v\n", http.StatusText(ise), err)
			w.WriteHeader(ise)
			w.Write([]byte(errorBody)) // "Internal Server Error: {error}"
		}
		return
	}
	newlineByte := []byte("\n")
	responseByte := append(jsonByte, newlineByte...)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	Respond(w, r, headers, responseByte, statusCode)
	return
} // end func RespondJSON

// RespondPlain is used for writing a Plain Text to the response body of incoming requests to our service
func RespondPlain(w http.ResponseWriter, r *http.Request, responseText string, statusCode int) {
	headers := make(map[string]string)
	headers["Content-Type"] = "text/plain"
	Respond(w, r, headers, []byte(responseText+"\n"), statusCode)
	return
} // end func RespondPlain

// RespondTemplateGcs is used for using a GCS template file for the response body of incoming requests to our service
func RespondTemplateGcs(w http.ResponseWriter, r *http.Request, bucketName, objectName, localFilename string, statusCode int, rdata ...any) {
	bodyBytes, found, foundName, size, updated, mime, err := GcsReadFileBytes(bucketName, objectName)
	contentSize := size
	if !found || size == 0 || err != nil {
		if !found || size == 0 {
			e("File Not Found in GCS: %s/%s -- Trying %s", bucketName, foundName, localFilename)
		} else if err != nil {
			e("GCS Read File Error: %+v -- Trying %s", err, localFilename)
		}

		// Try the localFilename
		bodyBytes, found, _, foundName, size, updated, mime, err = ReadFileBytes(localFilename)
		contentSize = size
		if !found || size == 0 || err != nil {
			e("Failed to Read the Local file %s: %+v", localFilename, err)
			NotFoundHandler(w, r)
			return
		}

		// Set the metadata
		contentType := mime
		contentDisposition := "inline; filename=" + PathBasename(localFilename)
		metadata := make(map[string]string)

		// Save the file to GCS
		sizeg, errg := GcsSaveFile(bucketName, objectName, bodyBytes, contentType, contentDisposition, metadata)
		if errg != nil {
			e("Failed to Copy the template file %s to GCS %s/%s: %+v", foundName, bucketName, objectName, errg)
		} else {
			o("Copied the template file %s to GCS %s/%s, %d bytes", foundName, bucketName, objectName, sizeg)
		}
	} // end if found

	// Set any placeholder values we might have in the template
	respondBytes := DynamicPlaceholdersPrefixBytes(bodyBytes, "", false, rdata...)
	if !bytes.Equal(respondBytes, bodyBytes) {
		contentSize = int64(len(respondBytes))
		//o("Serving content different than Template File %s", foundName)
		//o("%s: Content-Length: %d (template was %d)", foundName, contentSize, size)
		//o("%s: Last-Modified: %s (template was %s)", foundName, TimeHttpHeaderStr(TimeNow()), TimeHttpHeaderStr(updated))
		updated = TimeNow()
	}

	// Set the HTTP Headers
	//o("Content-Type: %s", mime)
	headers := make(map[string]string)
	if mime != "" {
		headers["Content-Type"] = mime
	}
	if contentSize > 0 {
		headers["Content-Length"] = s("%d", contentSize)
	}
	if updated != TimeZero() {
		headers["Last-Modified"] = TimeHttpHeaderStr(updated)
	}
	if StrContains(mime, "media") || StrContains(mime, "csv") {
		headers["Content-Disposition"] = s("inline; filename=\"%s\"", PathBasename(foundName))
	}

	// Read the Bytes to the Request Writer
	Respond(w, r, headers, respondBytes, statusCode)
	return
} // end func RespondTemplateGcs

// RespondTemplate is used for using a local template file for the response body of incoming requests to our service
func RespondTemplate(w http.ResponseWriter, r *http.Request, localFilename string, statusCode int, rdata ...any) {
	bodyBytes, found, _, foundName, size, updated, mime, err := ReadFileBytes(localFilename)
	contentSize := size
	if !found || size == 0 || err != nil {
		e("Failed to open Local file %s: %+v", localFilename, err)
		NotFoundHandler(w, r)
		return
	}

	// Set any placeholder values we might have in the template
	respondBytes := DynamicPlaceholdersPrefixBytes(bodyBytes, "", false, rdata...)
	if !bytes.Equal(respondBytes, bodyBytes) {
		contentSize = int64(len(respondBytes))
		//o("Serving content different than Template File %s", foundName)
		//o("%s: Content-Length: %d (template was %d)", foundName, contentSize, size)
		//o("%s: Last-Modified: %s (template was %s)", foundName, TimeHttpHeaderStr(TimeNow()), TimeHttpHeaderStr(updated))
		updated = TimeNow()
	}

	// Set the HTTP Headers
	//o("Content-Type: %s", mime)
	headers := make(map[string]string)
	if mime != "" {
		headers["Content-Type"] = mime
	}
	if contentSize > 0 {
		headers["Content-Length"] = s("%d", contentSize)
	}
	if updated != TimeZero() {
		headers["Last-Modified"] = TimeHttpHeaderStr(updated)
	}
	if StrContains(mime, "media") || StrContains(mime, "csv") {
		headers["Content-Disposition"] = s("inline; filename=\"%s\"", PathBasename(foundName))
	}

	// Read the Bytes to the Request Writer
	Respond(w, r, headers, respondBytes, statusCode)
	return
} // end func RespondTemplate

// Respond is a generic byte response writer for the response body of incoming requests to our service
func Respond(w http.ResponseWriter, r *http.Request, headers map[string]string, responseByte []byte, statusCode int) {
	okByte := []byte("OK\n")
	isOkByte := false
	if bytes.Equal(responseByte, okByte) {
		isOkByte = true
	}
	if len(headers) > 0 {
		for key, value := range headers {
			header := w.Header()
			_, found := header[key]
			if found {
				w.Header().Set(key, value)
				//o("Header Set [ %s: %s ]", key, value)
			} else {
				w.Header().Add(key, value)
				//o("Header Add [ %s: %s ]", key, value)
			}
		}
	}
	if !isOkByte {
		remoteAddr := r.RemoteAddr
		requestPath := r.URL.Path
		if StrContains(requestPath, "/x/mirror/") {
			o("%s [%d] Client %s: %s", requestPath, statusCode, remoteAddr, string(responseByte[:]))
		} else {
			o("%s [%d] Client %s", requestPath, statusCode, remoteAddr)
		}
	}
	w.WriteHeader(statusCode)
	w.Write(responseByte)
	return
} // end func Respond

func PublishToPubSubPool(projectID string, topicIDsPool []string, anyData any, attrinf map[string]interface{}) (msgId string, success bool, err error) {
	if projectID == "" {
		return "", false, eer("PubSub ProjectID is required")
	}
	var topicId string = ""
	topicsn := len(topicIDsPool)
	if topicsn == 0 {
		return "", false, eer("At least one PubSub Topic ID is required")
	} else if topicsn > 1 {
		for i := 0; i <= topicsn; i++ {
			if topicId == "" {
				tmax := AnyToInt64(topicsn) - 1
				var ii int64 = 0
				if tmax > 0 {
					ii = RandomInt(0, tmax)
				}
				topicId = topicIDsPool[ii]
			} // end if topicId
		} // end for i
		o("Publishing message to Topic %s from Pool %+v", topicId, topicIDsPool)
	} else {
		topicId = topicIDsPool[0]
	}
	if topicId == "" {
		return "", false, eer("PubSub TopicID is required")
	}
	return PublishToPubSub(projectID, topicId, anyData, attrinf)
} // end func PublishToPubSubPool

func PublishToPubSub(projectID string, topicID string, anyData any, attrinf map[string]interface{}) (id string, success bool, err error) {
	if projectID == "" {
		return "", false, eer("PubSub ProjectID is required")
	}
	if topicID == "" {
		return "", false, eer("PubSub TopicID is required")
	}

	id = ""
	success = false
	pubsub_start := time.Now()
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return "", false, eer("Failed to create client: %+v [%s]", err, Runtime(pubsub_start))
	}

	topic := client.Topic(topicID)
	attributes := make(map[string]string)
	data := AnyToByte(anyData)

	if len(attrinf) > 0 {
		for key, val := range attrinf {
			switch v := val.(type) {
			case string:
				attributes[key] = v
			default:
				vb := AnyToByte(v)
				attributes[key] = string(vb[:])
			}
		}
	}

	if len(data) == 0 {
		return "", false, eer("No Data to Publish: %s [%s]", string(data[:]), Runtime(pubsub_start))
	}

	// Publish a message.
	result := topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attributes,
	})

	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	id, err = result.Get(ctx)
	if err != nil {
		return "", false, eer("Failed to Get the PubSub Result: %+v [%s]", err, Runtime(pubsub_start))
	}

	//o("PublishToPubSub: Success! Published a message with a message ID: %s [%s]", id, Runtime(pubsub_start))
	success = true
	return id, true, nil
} // end func PublishToPubSub

func GetPubSubPushMessage(r *http.Request, subs ...string) (data map[string]interface{}, attributes map[string]interface{}, rx PubSubRequest, err error) {
	attributes = make(map[string]interface{})
	data = make(map[string]interface{})

	// Read the JSON Body
	err = GetRequestBodyJSON(r, &rx)
	if err != nil {
		return
	}

	// Optional: Validate the Subscription
	if len(subs) > 0 {
		rxSub := PathBasename(rx.Subscription)
		subFound := false
		for _, subscription := range subs {
			if rxSub == subscription || rxSub == subscription+"-sub" {
				subFound = true
			}
		} // end foreach sub
		if !subFound {
			err = er("Subscription %s doesn't match the required value: %+v", rx.Subscription, subs)
			e("GetPubSubPushMessage: %+v", err)
			rx.Message.Data = ""
			return
		}
	}

	// Set the Attributes
	attributes = rx.Message.Attributes

	// Decode the Message Data
	jsonb := make([]byte, 0)
	jsonb, err = base64.StdEncoding.DecodeString(rx.Message.Data)
	if err != nil {
		return
	}
	// Put the Decoded Data back in our rx Message
	jsonb = bytes.ReplaceAll(jsonb, []byte("\n"), []byte(" "))
	rx.Message.Data = string(jsonb[:])

	if len(jsonb) < 2 {
		err = er("No JSON Data Received from PubSub Message Data")
		e("GetPubSubPushMessage: %+v", err)
		return
	}

	// Try JSON Decoding the String
	err = JsonDecode(jsonb, &data)
	//o("GetPubSubPushMessage data: %+v", data)
	return
} // end func GetPubSubPushMessage

func GetAcnsClientNameFromDomain(d string) string {
	clientName := ""
	u := ParseUri(d)
	if u.Domain != "" {
		parts := StrSplit(u.Domain, ".")
		sub := parts[0]
		if sub != "" {
			parts = StrSplit(sub, "-")
			clientName = parts[0]
		}
	}
	return clientName
} // end func GetAcnsClientNameFromDomain

func GetTtsApiKey() string {
	if thisTtsApiKey != "" {
		return thisTtsApiKey
	}
	return thisApiKey
} // end func GetTtsApiKey

func GetTtsApiUrl(r *http.Request, tts_cmd ...string) string {
	var tts_url string = ""
	if thisTtsApiUrl != "" {
		tts_paths := append([]string{thisTtsApiUrl}, tts_cmd...)
		tts_url = PathJoin(tts_paths...)
	} else {
		tts_paths := append([]string{"/api/tts"}, tts_cmd...)
		tts_url = GetRequestBaseUrl(r, tts_paths...)
	}
	return tts_url
} // end func GetTtsApiUrl

func GetRedisApiKey() string {
	if thisRedisApiKey != "" {
		return thisRedisApiKey
	}
	return thisApiKey
} // end func GetRedisApiKey

func GetRedisApiUrl(r *http.Request, redis_cmd ...string) string {
	var redis_url string = ""
	if thisRedisApiUrl != "" {
		redis_paths := append([]string{thisRedisApiUrl}, redis_cmd...)
		redis_url = PathJoin(redis_paths...)
	} else {
		redis_paths := append([]string{"/api/redis-proxy"}, redis_cmd...)
		redis_url = GetRequestBaseUrl(r, redis_paths...)
	}
	return redis_url
} // end func GetRedisApiUrl

func GetDocsUrl(r *http.Request, docs_cmd ...string) string {
	docs_paths := append([]string{"/docs/"}, docs_cmd...)
	docs_paths = append([]string{thisApiServicePrefix}, docs_paths...)
	docs_url := GetRequestBaseUrl(r, docs_paths...)
	return docs_url
} // end func GetDocsUrl

func RedisDbConnect() (bool, error) {
	//o("thisRedisDbEnabled: %+v, thisRedisDbConnected: %+v", thisRedisDbEnabled, thisRedisDbConnected)
	if thisRedisDbEnabled && !thisRedisDbConnected {
		thisRedisDbConnected = true
		thisRedisDbError = nil

		// Global Context for Redis so we can use the connection in handlers and kill the connection on shutdown
		rdbCtx, rdbCtxExit = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		rdbOptions := &redis.Options{
			Addr:            redisHost + ":" + redisPort, // Redis address
			Password:        redisPass,
			DB:              0,
			MaxRetries:      3,
			MinRetryBackoff: 8 * time.Millisecond,
			MaxRetryBackoff: 512 * time.Millisecond,
			DialTimeout:     15 * time.Second,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			MinIdleConns:    1,
			PoolSize:        1000,
			PoolTimeout:     1 * time.Minute,
		}

		if thisRedisDbTlsEnabled {
			rdbOptions.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}

		// Global rdb Redis Client
		//o("rdbOptions: %+v", rdbOptions)
		rdb = redis.NewClient(rdbOptions)

		// Ping test the Redis Connection
		if err := RedisDbPing(); err != nil {
			e("Redis DB is Enabled, but Failed to connect to Redis Host %s:%s: %+v", redisHost, redisPort, err)
			thisRedisDbConnected = false // global variable
			thisRedisDbError = err       // global variable
		}

		if thisRedisDbConnected {
			o("Redis DB Client Connected Successfully! Redis DB Host: %s:%s", redisHost, redisPort)
			go RedisKeepAlivePing(rdb, 55*time.Second)
		}
	} // end if thisRedisDbEnabled
	return thisRedisDbConnected, thisRedisDbError
} // end func RedisDbConnect

func RedisDbDisconnect() {
	if thisRedisDbEnabled && thisRedisDbConnected {
		// Kill the RDB Connection
		thisRedisDbConnected = false
		defer rdbCtxExit()
	}
	return
} // end func RedisDbConnect

func RedisDbPing() error {
	redisPingCtx, redisPingCancel := context.WithTimeout(rdbCtx, 5*time.Second)
	defer redisPingCancel()

	err := rdb.Ping(redisPingCtx).Err()
	return err
} // end func RedisDbPing

func RedisKeepAlivePing(client *redis.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			if err := RedisDbPing(); err != nil {
				e("%+v", err)
			}
		}
	}
} // end func RedisKeepAlivePing

func CheckRedisDbRateLimit(contactUuid, rdbStrKey, rdbHashKey string, rateLimitMax, rdbKeyExpires int64) (canContinue bool, currentRateCount int64, err error) {
	currentRateCount = 1
	canContinue = true
	err = nil
	contactUuidDebug := ""
	if contactUuid != "" {
		contactUuidDebug = s(" for Contact [%s]", contactUuid)
	}

	// TODO TODO -- Add support for Redis Proxy API (microservice: /api/redis-proxy)

	if thisRedisDbEnabled && rateLimitMax > 0 {
		canContinue = false
		if !thisRedisDbConnected {
			thisRedisDbConnected, thisRedisDbError = RedisDbConnect()
		}
		if thisRedisDbError != nil {
			err = er("Redis DB enabled but there was an error: %+v", thisRedisDbError)
			return
		} else if !thisRedisDbConnected {
			err = er("Redis DB enabled but not connected.")
			return
		} // end if thisRedisDbConnected

		var rdbGetRx RedisGetRequest
		var rdbKeyFound bool = false
		redisNil := MakeRedisNil() // This is the error returned for Redis GET not found
		rdbGetRx.Key = rdbHashKey
		rdbGetRp, rdbErr := RedisGet(rdbGetRx)
		if rdbErr == nil {
			// Key found - Check if we've gone over the CPS limit this minute
			rdbKeyFound = true
			rdbGetResult := rdbGetRp.GetValue()
			if rdbGetResult != nil {
				currentRateCountStr, ok := rdbGetResult.(string)
				if ok {
					currentRateCount = AnyToInt64(currentRateCountStr)
				} else {
					// RDB Counter value is not an int64 -- debug what we got
					o("Got a different value type (%T) from the Redis DB GET%s, trying to convert to int64: %+v", rdbGetResult, contactUuidDebug, rdbGetResult)
					currentRateCount = AnyToInt64(rdbGetResult)
				} // end if ok
			} // end if rdbGetResult
		} else if rdbErr != redisNil {
			// Some RDB error -- maybe the connection failed?
			err = er("Got an Error from the Redis DB getting the Rate Limit%s, RDB Error: %+v", contactUuidDebug, rdbErr)
			return
		} // end if rdbGetRp

		if currentRateCount > rateLimitMax {
			// Rate Limit exceeded NACK this request
			err = er("RDB Rate Limit Exceeded%s! Redis DB Rate Limit (%d/%d)", contactUuidDebug, currentRateCount, rateLimitMax)
			return
		} // end if currentRateCount
		canContinue = true
		err = nil

		// Increment the existing key
		var rdbIncrRx RedisIncrRequest
		rdbIncrRx.Key = rdbHashKey
		rdbIncrRx.Value = 1               // Increment by 1
		rdbIncrRx.Expires = rdbKeyExpires // In Seconds
		rdbIncrRx.DoExpire = false
		if rdbKeyExpires > 0 && !rdbKeyFound {
			rdbIncrRx.DoExpire = true // We only want to set the expire if the key was not found
		}
		rdbIncrRp, rdbErr := RedisIncr(rdbIncrRx)
		if !rdbIncrRp.Success || rdbErr != nil {
			e("Got an Error incrementing the RDB Rate Limit%s: %+v", contactUuidDebug, rdbErr)
		} else {
			currentRateCount = rdbIncrRp.Value
			o("RDB Rate Limit Counter is now (%d/%d)%s at Redis DB Key = %s", currentRateCount, rateLimitMax, contactUuidDebug, rdbStrKey)
		}

	} // end if thisRedisDbEnabled
	return
} // end func CheckRedisDbRateLimit

func RedisRestApiGet(r *http.Request, redis_get_key string) (RedisGetResponse, error) {
	var redisResp RedisGetResponse
	redisResp.Success = false
	start := time.Now()
	if redis_get_key == "" {
		return redisResp, eer("No Redis DB Key Specified")
	}
	redis_url := GetRedisApiUrl(r, "/get/", redis_get_key)
	if redis_url == "" {
		return redisResp, eer("No Redis API URL Configured")
	}
	headers := make(map[string]string)
	headers["x-api-key"] = GetRedisApiKey()
	headers["Accept"] = "application/json"
	api_results, err := NewRestApiRequest(http.MethodGet, redis_url, &redisResp, headers, 10, 0)
	redisResp.SetRuntime(RuntimeSecondsStr(start))
	if err != nil {
		return redisResp, err
	}
	redis_code := api_results.GetCode()
	if redis_code == 404 {
		// Not Found
		redisResp.Success = false
		return redisResp, eer("Redis Key NOT FOUND in DB")
	} else if redis_code != 200 {
		// Some other Error
		redisResp.Success = false
		return redisResp, eer("Redis API Request Failed")
	} // end if redis

	// Successful
	return redisResp, nil
} // end func RedisRestApiGet

func RedisDel(rx RedisDelRequest) (resp RedisDelResponse, err error) {
	if rx.Key == "" {
		err = er("Key Missing, Required")
		return
	}

	resp.Key = rx.Key
	resp.Success = true
	resp.SetRuntime("0s")
	start := TimeNow()

	// Handle the Redis Query & Result
	//o("Key: %s", rx.Key)
	err = rdb.Del(rdbCtx, rx.Key).Err()
	resp.SetRuntime(RuntimeSecondsStr(start))

	// Success or Failure
	return
} // end func RedisDel

func RedisGet(rx RedisGetRequest) (resp RedisGetResponse, err error) {
	if rx.Key == "" {
		err = er("Key Missing, Required")
		return
	}

	resp.Key = rx.Key
	resp.Success = true
	resp.SetRuntime("0s")
	start := TimeNow()

	// Handle the Redis Query & Result
	//o("Key: %s", redisKey)
	value, err := rdb.Get(rdbCtx, rx.Key).Result()
	resp.SetRuntime(RuntimeSecondsStr(start))
	if err != nil {
		// Not Found or Error
		return
	} else {
		// Found
		resp.SetValue(value)
	}

	// Successful
	return
} // end func RedisGet

func RedisIncr(rx RedisIncrRequest) (resp RedisIncrResponse, err error) {
	if rx.Key == "" {
		err = er("Key Missing, Required")
		return
	}

	if rx.Value == 0 {
		rx.Value = 1
	}

	resp.SetRx(rx)
	resp.Success = true
	resp.SetRuntime("0s")
	start := TimeNow()

	// Handle the Redis Query & Result
	//o("Key: %s, Value: %+v, Expires: %+v", rx.Key, rx.Value, rx.Expires)
	result, err := rdb.IncrBy(rdbCtx, rx.Key, rx.Value).Result()
	if err != nil {
		resp.Success = false
	} else {
		resp.Value = result
	}

	if err == nil && rx.DoExpire {
		err = rdb.Expire(rdbCtx, rx.Key, TimeDuration(rx.Expires)).Err()
	}

	// Success or Failure
	resp.SetRuntime(RuntimeSecondsStr(start))
	return
} // end func RedisIncr

func RedisSet(rx RedisSetRequest) (resp RedisSetResponse, err error) {
	if rx.Key == "" {
		err = er("Key Missing, Required")
		return
	}

	resp.SetRx(rx)
	resp.Success = true
	resp.SetRuntime("0s")
	start := TimeNow()

	// Handle the Redis Query & Result
	//o("Key: %s, Value: %+v, Expires: %+v", rx.Key, rx.Value, rx.Expires)
	if rx.Expires == 0 {
		err = rdb.Set(rdbCtx, rx.Key, rx.Value, 0).Err()
	} else {
		err = rdb.Set(rdbCtx, rx.Key, rx.Value, TimeDuration(rx.Expires)).Err()
	}
	if err != nil {
		resp.Success = false
	}
	resp.SetRuntime(RuntimeSecondsStr(start))

	// Success or Failure
	return
} // end func RedisSet

func BasicAuth(username, password string) string {
	auth := username
	if password != "" {
		auth += ":" + password
	}
	r := Base64EncodeStr(auth)
	return r
} // end func BasicAuth

func Base64DecodeByte(b []byte) []byte {
	mn := base64.StdEncoding.DecodedLen(len(b))
	decodedb := make([]byte, mn)
	dn, err := base64.StdEncoding.Decode(decodedb, b)
	if err != nil {
		e("Base64DecodeByte: Error: %+v", err)
		return b
	}
	r := decodedb[:dn]
	return r
} // end func Base64DecodeByte

func Base64DecodeStr(str string) string {
	b, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		e("Base64DecodeStr: Error: %+v", err)
		return str
	}
	r := string(b[:])
	return r
} // end func Base64DecodeStr

func Base64EncodeByte(b []byte) []byte {
	str := base64.StdEncoding.EncodeToString(b)
	return []byte(str)
} // end func Base64EncodeByte

func Base64EncodeStr(str string) string {
	str = base64.StdEncoding.EncodeToString([]byte(str))
	return str
} // end func Base64EncodeStr

func BytesGzipped(b []byte) bool {
	if len(b) >= 3 {
		if b[0] == 2 && b[1] == 31 && b[2] == 8 {
			return true
		}
	}
	return false
} // end func BytesGzipped

func UrlDecodeStr(str string) string {
	strr, err := url.QueryUnescape(str)
	if err != nil {
		e("UrlDecodeStr: Error: %+v", err)
		return str
	}
	strr = strings.TrimSpace(strr)
	return strr
} // end func UrlDecodeStr

func UrlEncodeStr(str string) string {
	str = url.QueryEscape(strings.TrimSpace(str))
	return str
} // end func UrlEncodeStr

func ValidateBearerFormat(bearerToken string) (string, bool) {
	bearerTokenValid := false
	if bearerToken != "" && strings.Contains(bearerToken, " ") == true {
		bearerSplit := StrSplit(bearerToken, " ")
		if len(bearerSplit) == 2 {
			if bearerSplit[0] == "Bearer" {
				bearerToken = bearerSplit[1] // "Bearer {BEARER_TOKEN}"
				if bearerToken != "" {
					bearerTokenValid = true
				}
			}
		}
	} else if bearerToken != "" {
		bearerTokenValid = true // "{BEARER_TOKEN}"
	} // end if bearerToken
	return bearerToken, bearerTokenValid
} // end function ValidateBearerFormat

func CheckAuthedBearerTokens(bearerToken string) (auth bool, found bool) {
	auth, found = thisAuthedBearerTokens[bearerToken]
	return
} // end func CheckAuthedBearerTokens

func CheckApiAuthKey(r *http.Request) (requestAuth bool, err error) {
	requestAuth = false
	err = nil
	rx_api_key := r.Header.Get("x-api-key")
	if rx_api_key == "" {
		rx_api_key = r.Header.Get("X-Api-Key")
	}
	if rx_api_key == "" {
		rx_api_key = r.Header.Get("Authorization")
	}
	if rx_api_key == thisApiKey {
		requestAuth = true
	} else if rx_api_key == "" {
		err = er("API Key Required")
	} else if rx_api_key != thisApiKey {
		err = er("API Key Not Valid")
	}
	return
} // end func CheckApiAuthKey

func CheckApiAuthKeyOrBearer(r *http.Request, allowedAuthEmails []string) (requestAuth bool, err error) {
	requestAuth, err = CheckApiAuthKey(r)
	erm := ""
	if err != nil || requestAuth == false {
		erm = s("API Key Not Found or Matched. %+v", err)
		requestAuth, err = CheckApiAuthBearer(r, allowedAuthEmails)
		if err != nil {
			e("API Bearer Auth Not Found or Matched. %+v", err)
		}
	} // end if requestAuth
	if requestAuth == false {
		e("%s", erm)
	}
	return
} // end func CheckApiAuthKeyOrBearer

func CheckApiAuthBearer(r *http.Request, allowedAuthEmails []string) (requestAuth bool, err error) {
	// Check the GCP Bearer Key
	bearerToken := r.Header.Get("Authorization") // Authorization: Bearer {BEARER_TOKEN}
	if bearerToken == "" {
		bearerToken = r.Header.Get("authorization")
	}
	if bearerToken == "" {
		bearerToken = r.Header.Get("Authentication")
	}
	if bearerToken == "" {
		bearerToken = r.Header.Get("authentication")
	}
	if bearerToken != "" {
		requestAuth = CheckGoogleTokenInfo(bearerToken, allowedAuthEmails)
	}
	if requestAuth == false {
		err = er("Bearer Token is Missing or Invalid")
	}
	return
} // end func CheckApiAuthBearer

func CheckGoogleTokenInfo(bearerToken string, allowedAuthEmails []string) bool {
	// Check the Bearer Key
	var gti GoogleTokenInfo
	gti_start := time.Now()
	bearerAuth := false
	bearerToken, bearerTokenValid := ValidateBearerFormat(bearerToken)
	bearerAuth, found := CheckAuthedBearerTokens(bearerToken)
	if found {
		o("Token already checked and found to be %+v", bearerAuth)
		return bearerAuth
	}
	if bearerTokenValid {
		// rp := GET https://oauth2.googleapis.com/tokeninfo?id_token={BEARER_TOKEN}
		rp, err := NewRestApiRequest(http.MethodGet, "https:/"+"/oauth2.googleapis.com/tokeninfo?id_token="+bearerToken, &gti, 10, 0)
		if err != nil {
			e("GET Request got an error: %+v", err)
			return false
		}
		if rp.GetCode() == 0 {
			rp.SetCode(401)
		}
		//o("[%s]: %d Response: %+v and GoogleTokenInfo: %+v", Runtime(gti_start), rp.GetCode(), rp, gti)
		if rp.GetCode() == 200 && StrContains(gti.Iss, "accounts.google.com") == true && gti.EmailVerified == "true" {
			if len(allowedAuthEmails) > 0 {
				for _, auth_email := range allowedAuthEmails {
					if StrContains(gti.Email, auth_email) && StrContains(auth_email, gti.Email) { // This is a case-insensitive equals
						bearerAuth = true
					}
				}
				if bearerAuth {
					o("Authorized [%s]: GoogleTokenInfo.Email: %s", Runtime(gti_start), gti.Email)
				}
			}
		}
		if !bearerAuth {
			e("Not Authorized [%s]: GoogleTokenInfo Resp: %+v", Runtime(gti_start), rp)
		}
		thisAuthedBearerTokens[bearerToken] = bearerAuth
	} else if bearerToken != "" {
		e("Not Authorized [%s], Token is invalid! %s", Runtime(gti_start), bearerToken)
	} else {
		e("Not Authorized [%s], Token is empty!", Runtime(gti_start))
	} // end if bearerToken
	return bearerAuth
} // end func CheckGoogleTokenInfo

func CheckPhoneNumber(input string, opts ...bool) bool {
	allowShortCode := false
	if len(opts) > 0 {
		allowShortCode = opts[0]
	}
	isShortCode := false
	isValid := false
	strlen := len(input)
	if allowShortCode == true && strlen < 10 {
		isShortCode = true
		isValid = true
	} else if strlen >= 10 && strlen <= 25 {
		isValid = true
	}
	if isValid {
		newMethodChecked := false
		if len(thisCheckPhoneNumberCountries) == 0 {
			thisCheckPhoneNumberCountries = []string{"US"}
		}
		// New Method - use the library
		for _, regionCode := range thisCheckPhoneNumberCountries {
			thisPhoneNumber, err := phonenumbers.Parse(input, regionCode)
			if err == nil {
				newMethodChecked = true
				if isShortCode {
					isValid = phonenumbers.IsValidShortNumberForRegion(thisPhoneNumber, regionCode)
				} else {
					isValid = phonenumbers.IsValidNumberForRegion(thisPhoneNumber, regionCode)
				}
				if isValid {
					return isValid
				} // end if isValid
			} // end if err
		} // end foreach regionCode

		if !newMethodChecked {
			// Old Method - regex (not very good --- +1234567890 passes)
			re := regexp.MustCompile(`^\+?(?:(?:\(?(?:00|\+)([1-4]\d\d|[1-9]\d?)\)?)?[\-\.\ \\\/]?)?((?:\(?\d{1,}\)?[\-\.\ \\\/]?){0,})(?:[\-\.\ \\\/]?(?:#|ext\.?|extension|x)[\-\.\ \\\/]?(\d+))?$`)
			isValid = re.MatchString(input)
		} // end if newMethodChecked
	} // end if isValid

	if isValid == false {
		o("CheckPhoneNumber('%s') is not a phone number", input)
	}
	return isValid
} // end func CheckPhoneNumber

func FormatPhoneNumber(input string) string {
	if CheckPhoneNumber(input) {
		first := SubStr(input, 0, 1)
		if first != "+" && first != "1" && len(input) == 10 {
			if CheckPhoneNumber("1" + input) {
				input = "1" + input
			}
		}
		if first != "+" {
			input = "+" + input
		}
	}
	return input
} // end func FormatPhoneNumber

func JsonEncode(jsonObj any) (jsonByte []byte, err error) {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	jsonByte = make([]byte, 0)

	// Check for Data Cycles (interface/pointers)
	err = acyclic.Check(jsonObj)
	if err != nil {
		e("%+v", err)
		acyclic.Print(jsonObj)
		return
	}

	// JSON Encode the Data
	jsonByte, err = json.Marshal(jsonObj)
	if err != nil {
		e("%+v", err)
		acyclic.Print(jsonObj)
		return
	}
	return
} // end func JsonEncodeBytes

func JsonEncodeStr(jsonObj any) (string, error) {
	jsonByte, err := JsonEncode(jsonObj)
	return string(jsonByte[:]), err
} // end func JsonEncodeStr

func JsonEncodeStrOut(jsonObj any) string {
	str, _ := JsonEncodeStr(jsonObj)
	return str
} // end func JsonEncodeStrOut

func JsonDecode[T any](jsonIn any, jsonObj *T) error {
	defer RecoverErrorStack(TraceFile()) // Panic Error handling wrapped
	var err error
	bStr := make([]byte, 0)
	switch j := jsonIn.(type) {
	case string:
		bStr = []byte(j)
	case []byte:
		bStr = j
	case T:
		jsonObj = jsonIn.(*T)
	case map[string]interface{}:
		bStr, err = JsonEncode(j)
		if err != nil {
			return eer("%+v", err)
		}
	case []interface{}:
		bStr, err = JsonEncode(j)
		if err != nil {
			return eer("%+v", err)
		}
	default:
		return eer("Input Type %T is not supported!", j)
	} // end if
	if len(bStr) == 0 {
		return eer("Input is empty!")
	}
	err = json.Unmarshal(bStr, &jsonObj)
	return err
} // end func JsonDecode

///////////////////////////////////////////////////////////////////////////////////
//////// REST API Response Handlers
///////////////////////////////////////////////////////////////////////////////////

func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	iconFile := thisPageFavicon
	if iconFile == "" {
		iconFile = "assets/favicon.png"
	}
	exists, _, err := PathExists(iconFile)
	if exists == true && err == nil {
		http.ServeFile(w, r, iconFile)
	} else {
		NotFoundHandler(w, r)
	}
} // end func FaviconHandler

func PublicFileHandler(w http.ResponseWriter, r *http.Request) {
	isDocsPublic := false
	requestPath := r.URL.Path
	requestPathLen := len(requestPath)
	//o("RequestPath: %s", requestPath)
	if requestPathLen > 3 {
		if SubStr(requestPath, -3) == ".go" || StrContains(requestPath, "/private/") || StrContains(requestPath, "../") {
			e("Rejected RequestPath: %s", requestPath)
			ForbiddenHandler(w, r)
			return
		}
	}
	publicPrefix := "public"
	if requestPathLen > thisApiServicePrefixLen {
		//o("RequestPath: %s, Checking Prefix: %s ?? %s", requestPath, thisApiServicePrefix, SubStr(requestPath, 0, thisApiServicePrefixLen))
		if SubStr(requestPath, 0, thisApiServicePrefixLen) == thisApiServicePrefix {
			requestPath = SubStr(requestPath, thisApiServicePrefixLen)
			requestPathLen = len(requestPath)
			//o("Adjusted RequestPath: %s", requestPath)
		}
	}

	if SubStr(requestPath, 0, 5) == "/api/" {
		requestPath = SubStr(requestPath, 4)
		requestPathLen = len(requestPath)
		//o("Adjusted RequestPath: %s", requestPath)
	}

	//o("Checking for /async/: %s -OR- /direct/: %s", SubStr(requestPath, 0, 7), SubStr(requestPath, 0, 8))
	if SubStr(requestPath, 0, 7) == "/async/" {
		requestPath = SubStr(requestPath, 6)
		requestPathLen = len(requestPath)
		//o("Adjusted RequestPath: %s", requestPath)
	} else if SubStr(requestPath, 0, 8) == "/direct/" {
		requestPath = SubStr(requestPath, 7)
		requestPathLen = len(requestPath)
		//o("Adjusted RequestPath: %s", requestPath)
	}

	//o("Checking for /docs/public/: %s -OR- /docs/: %s", SubStr(requestPath, 0, 13), SubStr(requestPath, 0, 6))
	if SubStr(requestPath, 0, 13) == "/docs/public/" {
		requestPath = SubStr(requestPath, 13)
		requestPathLen = len(requestPath)
		//o("Adjusted RequestPath: %s", requestPath)
		publicPrefix = "docs/public"
		isDocsPublic = true
	} else if SubStr(requestPath, 0, 6) == "/docs/" {
		requestPath = SubStr(requestPath, 6)
		requestPathLen = len(requestPath)
		//o("Adjusted RequestPath: %s", requestPath)
		publicPrefix = "docs/public"
		isDocsPublic = true
	}
	if requestPath != r.URL.Path {
		o("Adjusted RequestPath from: %s to: %s", r.URL.Path, requestPath)
	}

	// Join the request path with the local public path
	publicFile := PathJoin(publicPrefix, requestPath)
	//o("Checking: %s", publicFile)
	exists, isDirectory, err := PathExists(publicFile)
	if isDocsPublic && (exists != true || err != nil) {
		publicFile = s("%s%s", "docs", requestPath)
		o("Checking: %s", publicFile)
		exists, isDirectory, err = PathExists(publicFile)
	}
	if exists == true && err == nil && isDirectory == true {
		indexPath := "index.html"
		//o("The last character of %s is %s", publicFile, string(publicFile[len(publicFile)-1:]))
		if string(publicFile[len(publicFile)-1:]) != "/" {
			indexPath = "/" + indexPath
		}
		indexExists, _, indexErr := PathExists(publicFile + indexPath)
		if indexExists == true && indexErr == nil {
			exists = true
			isDirectory = false
			err = nil
			publicFile = publicFile + indexPath
		}
	}
	if publicFile != "public" && publicFile != "public/" && exists == true && err == nil {
		if isDirectory {
			o("Serving Directory: %s", publicFile)
		} else {
			o("Serving File: %s", publicFile)
		}
		http.ServeFile(w, r, publicFile)
	} else if r.URL.Path == "/" {
		ApiDocsRedirector(w, r)
	} else {
		o("File Not Found: %s - Redirecting to Docs", r.URL.Path)
		ApiDocsRedirector(w, r)
	}
	return
} // end func PublicFileHandler

func ApiRequestAcceptedHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusAccepted, opt...)
	return
} // end func ApiRequestAcceptedHandler

func ApiRequestOkHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusOK, opt...)
	return
} // end func ApiRequestOkHandler

func BadRequestHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusBadRequest, opt...)
	return
} // end func BadRequestHandler

func InternalServerErrorHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusInternalServerError, opt...)
	return
} // end func InternalServerErrorHandler

func ForbiddenHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusForbidden, opt...)
	return
} // end func ForbiddenHandler

func NotFoundHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusNotFound, opt...)
	return
} // end func NotFoundHandler

func TooEarlyHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusTooEarly, opt...)
	return
} // end func TooEarlyHandler

func UnauthorizedHandler(w http.ResponseWriter, r *http.Request, opt ...any) {
	StatusResponseHandler(w, r, http.StatusUnauthorized, opt...)
	return
} // end func UnauthorizedHandler

func StatusResponseHandler(w http.ResponseWriter, r *http.Request, respCode int, opt ...any) {
	var resp StatusResponse
	codeMessage := http.StatusText(respCode)
	docsUrl := GetDocsUrl(r)
	resp.SetCode(respCode)
	resp.SetMessage(codeMessage)
	if len(opt) > 0 {
		for _, op := range opt {
			switch input := op.(type) {
			case string:
				if len(input) > 0 {
					isDocsUrl := StrContains(input, docsUrl)
					if resp.GetDocsUrl() == "" && isDocsUrl {
						resp.SetDocsUrl(input)
					} else if resp.GetStatusUrl() == "" && !isDocsUrl && (SubStr(input, 0, 4) == "http" || SubStr(input, 0, 4) == "wss:") {
						resp.SetStatusUrl(input)
					} else if resp.GetPath() == "" && SubStr(input, 0, 1) == "/" {
						resp.SetPath(input)
					} else if resp.GetRuntime() == "" && IsRuntimeStr(input) == true {
						resp.SetRuntime(input)
					} else if resp.GetMessage() == codeMessage {
						resp.AppendMessage(": " + input)
					} else {
						resp.AppendMessage(input)
					}
				}
			case []interface{}:
				if len(input) > 0 {
					resp.AppendData(input...)
				} // end if input
			case map[string]interface{}:
				if len(input) > 0 {
					resp.AppendData(input)
				} // end if input
			case time.Time:
				// Assume this is a Start Time
				if resp.GetRuntime() == "" {
					resp.SetRuntime(Runtime(input))
				} else if resp.GetDatetime() == "" {
					resp.SetDatetime(input)
				}
			case time.Duration:
				// Assume this is a Runtime Duration
				if resp.GetRuntime() == "" {
					resp.SetRuntime(RuntimeDuration(input))
				}
			case float32:
				// Assume this is a Runtime Duration in Seconds
				if resp.GetRuntime() == "" {
					resp.SetRuntime(RuntimeFloat64(float64(input)))
				}
			case float64:
				// Assume this is a Runtime Duration in Seconds
				if resp.GetRuntime() == "" {
					resp.SetRuntime(RuntimeFloat64(input))
				}
			case int, int32, int64, uint, uint32, uint64:
				// Assume this is Total Rows
				resp.SetTotalRows(AnyToInt64(input))
			case error:
				resp.AppendMessage(s(": Error: %+v", input))
			default:
				v := reflect.ValueOf(input)
				switch v.Kind() {
				case reflect.Slice, reflect.Array:
					sliceLen := 0
					if v.Type().Kind() == reflect.Slice && v.Len() > 0 {
						sliceLen = v.Len()
					} else if v.Type().Kind() == reflect.Array && v.Type().Len() > 0 {
						sliceLen = v.Type().Len()
					}
					if sliceLen > 0 {
						ds := make([]interface{}, 0)
						for i := 0; i < sliceLen; i++ {
							ds = append(ds, v.Index(i).Interface())
						}
						resp.AppendData(ds...)
					} // end if sliceLen
				case reflect.Map:
					resp.AppendData(v.Interface())
				case reflect.Struct:
					data := StructToMapStringInterface(input)
					if len(data) > 0 {
						resp.AppendData(data)
					}
				}
			} // end switch input
		} // end foreach input
	} // end if opt
	if respCode != http.StatusOK && respCode != http.StatusAccepted {
		if resp.GetDocsUrl() == "" {
			resp.SetDocsUrl(docsUrl)
		}
	}
	RespondJSON(w, r, respCode, resp)
	return
} // end func StatusResponseHandler

func BadRequestXmlHandler(w http.ResponseWriter, r *http.Request, isXml bool, xmlTemplateGcs, xmlTemplate string, opt ...any) {
	defer RecoverErrorStackRequest(w, r, TraceFile()) // Panic Error handling wrapped
	if isXml {
		// XML Response - Rejected
		RespondTemplateGcs(w, r, thisGcsStorageBucket, xmlTemplateGcs, xmlTemplate, http.StatusOK, opt...)
		return
	} else {
		// JSON Response
		opt = append(opt, GetDocsUrl(r))
		BadRequestHandler(w, r, opt...)
		return
	}
} // end func BadRequestXmlHandler

func InternalServerErrorXmlHandler(w http.ResponseWriter, r *http.Request, isXml bool, xmlTemplateGcs, xmlTemplate string, opt ...any) {
	defer RecoverErrorStackRequest(w, r, TraceFile()) // Panic Error handling wrapped
	if isXml {
		// XML Response - Rejected
		RespondTemplateGcs(w, r, thisGcsStorageBucket, xmlTemplateGcs, xmlTemplate, http.StatusOK, opt...)
		return
	} else {
		// JSON Response
		opt = append(opt, GetDocsUrl(r))
		InternalServerErrorHandler(w, r, opt...)
		return
	}
} // end func InternalServerErrorXmlHandler

func NotFoundXmlHandler(w http.ResponseWriter, r *http.Request, isXml bool, xmlTemplateGcs, xmlTemplate string, opt ...any) {
	defer RecoverErrorStackRequest(w, r, TraceFile()) // Panic Error handling wrapped
	if isXml {
		// XML Response - Rejected
		RespondTemplateGcs(w, r, thisGcsStorageBucket, xmlTemplateGcs, xmlTemplate, http.StatusOK, opt...)
		return
	} else {
		// JSON Response
		opt = append(opt, GetDocsUrl(r))
		NotFoundHandler(w, r, opt...)
		return
	}
} // end func NotFoundXmlHandler

// HealthCheckHandler returns an OK plain text endpoint
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	RespondPlain(w, r, "OK", http.StatusOK)
} // end func HealthCheckHandler

// HealthCheckHandler returns an OK plain text endpoint
func RedisDbHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	//o("thisRedisDbEnabled: %v, thisRedisDbConnected: %v, thisRedisDbError: %v", thisRedisDbEnabled, thisRedisDbConnected, thisRedisDbError)
	if thisRedisDbEnabled {
		if !thisRedisDbConnected {
			thisRedisDbConnected, thisRedisDbError = RedisDbConnect()
			//o("Retry: thisRedisDbEnabled: %v, thisRedisDbConnected: %v, thisRedisDbError: %v", thisRedisDbEnabled, thisRedisDbConnected, thisRedisDbError)
		}
		if thisRedisDbError != nil {
			msg := s("Redis DB enabled but there was an error: %+v", thisRedisDbError)
			InternalServerErrorHandler(w, r, msg)
			return
		} else if !thisRedisDbConnected {
			msg := s("Redis DB enabled but not connected.")
			InternalServerErrorHandler(w, r, msg)
			return
		} // end if thisRedisDbEnabled
		RespondPlain(w, r, "OK", http.StatusOK)
	} else {
		err := eer("Redis DB is Disabled.")
		BadRequestHandler(w, r, err)
	} // end if thisRedisDbEnabled
} // end func RedisDbHealthCheckHandler

// MirrorRequestHandler is a debug endpoint that responds with information about the request received
//
// @Id          MirrorRequestHandler
// @Tags        X-Debug
// @Summary     A debug endpoint that responds with information about the request received
// @Accept      json
// @Produce     json
// @Param       id path string true "An example Path Parameter ID"
// @Param       request body MirrorRequest true "Client Request to be Mirrored."
// @Success     200 {object} StatusResponse
// @Failure     401 {object} StatusResponse
// @Router      /x/mirror/{id}/ [post]
// @Security    Bearer || Key
func MirrorRequestHandler(w http.ResponseWriter, r *http.Request) {
	defer RecoverErrorStackRequest(w, r, TraceFile()) // Panic Error handling wrapped
	rx_start := TimeNow()

	// Check the API Authentication
	rp := make(map[string]interface{})
	requestAuth, requestAuthErr := CheckApiAuthKeyOrBearer(r, thisGoogleTokenAuthEmails)
	rp["authorized"] = requestAuth
	uri := GetRequestUri(r, r.URL.Path)
	rp["uri"] = uri
	docsUrl := GetDocsUrl(r)
	headers, err := AnyToHeadersMap(r.Header)
	if err != nil {
		e("No Headers Mapped (%T): %+v, Error: %+v", r.Header, r.Header, err)
		rp["headers"] = r.Header
	} else if len(headers) > 0 {
		rp["headers"] = headers
	} else {
		o("No Headers Mapped (%T): %+v", r.Header, r.Header)
		rp["headers"] = r.Header
	}
	filter := []string{""}
	rx, err := MapFormParams(r, filter...)
	if err != nil && len(rx) > 0 {
		rp["rx"] = rx
	}
	id := GetRequestIdFromPath(r.URL.Path, "mirror", thisApiServicePrefix)
	if id != "" {
		rp["id"] = id
	}

	var x StatusResponse
	if requestAuth {
		x.SetCode(http.StatusOK)
	} else {
		x.SetCode(http.StatusUnauthorized)
	}
	x.SetMessage(s("This is an example at %s", NowDatetime()))
	x.SetData(rx)
	x.SetRuntime(rx_start)
	x.SetDocsUrl(docsUrl)
	rp["statusResponse"] = x
	//o("[%d] %s %s", x.GetCode(), r.URL.Path, JsonEncodeStrOut(rp))
	if requestAuth {
		ApiRequestOkHandler(w, r, rp, Runtime(rx_start), x.GetMessage())
	} else {
		UnauthorizedHandler(w, r, rp, Runtime(rx_start), requestAuthErr)
	}
	return
} // end func MirrorRequestHandler

// VersionCheckHandler returns a JSON version endpoint
func VersionCheckHandler(w http.ResponseWriter, r *http.Request) {
	var resp VersionResponse
	resp.Version = thisAppVersion
	RespondJSON(w, r, http.StatusOK, resp)
} // end func VersionCheckHandler

// ApiDocsRedirector redirects to the /api/{service}/docs URL
func ApiDocsRedirector(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, GetDocsUrl(r), http.StatusSeeOther)
	return
} // end func ApiDocsRedirector

// ApiDocsHandler is the /api/{service}/docs handler
func ApiDocsHandler(w http.ResponseWriter, r *http.Request) {
	httpswag.WrapHandler(w, r)
	return
} // end func ApiDocsHandler

///////////////////////////////////////////////////////////////////////////////////
//////// Google Cloud Storage Functions
///////////////////////////////////////////////////////////////////////////////////

func GcsCopyFile(srcBucket string, srcObjectName string, dstBucket string, dstObjectName string) (bool, error) {
	if SubStr(srcObjectName, 0, 1) == "/" {
		srcObjectName = SubStr(srcObjectName, 1)
	}
	if SubStr(dstObjectName, 0, 1) == "/" {
		dstObjectName = SubStr(dstObjectName, 1)
	}
	if srcBucket == "" || srcObjectName == "" || dstBucket == "" || dstObjectName == "" {
		return false, eer("Missing Required Params: %s, %s, %s, %s", srcBucket, srcObjectName, dstBucket, dstObjectName)
	}
	start_copy := time.Now()
	exists, size, _, err := GcsFileExists(srcBucket, srcObjectName)
	if err != nil {
		return false, err
	} else if exists != true {
		return false, nil // File Not Found
	} else if size == 0 {
		return false, eer("Source File size is nil")
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return false, err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	src := client.Bucket(srcBucket).Object(srcObjectName)
	dst := client.Bucket(dstBucket).Object(dstObjectName)

	// Optional: set a generation-match precondition to avoid potential race
	// conditions and data corruptions. The request to copy is aborted if the
	// object's generation number does not match your precondition.
	// For a dst object that does not yet exist, set the DoesNotExist precondition.
	dst = dst.If(storage.Conditions{DoesNotExist: true})
	// If the destination object already exists in your bucket, set instead a
	// generation-match precondition using its generation number.
	// attrs, err := dst.Attrs(ctx)
	// if err != nil {
	//      return er("object.Attrs: %w", err)
	// }
	// dst = dst.If(storage.Conditions{GenerationMatch: attrs.Generation})
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return false, err
	}

	o("File copied %s/%s to %s/%s [%s]", srcBucket, srcObjectName, dstBucket, dstObjectName, Runtime(start_copy))
	return true, nil
} // end func GcsCopyFile

func GcsDeleteFile(bucketName string, objectName string) (bool, error) {
	if bucketName == "" {
		bucketName = thisGcsStorageBucket // try the default
	}
	if bucketName == "" {
		return false, eer("No GCS Bucket provided")
	}
	if SubStr(objectName, 0, 1) == "/" {
		objectName = SubStr(objectName, 1)
	}
	if objectName == "" {
		return false, eer("No GCS Object Filename provided")
	}

	// Create the Storage Bucket Client
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return false, err
	}
	defer storageClient.Close()

	// Connect to the Bucket
	bucket := storageClient.Bucket(bucketName)

	// Check if the File Exists
	obj := bucket.Object(objectName)
	_, err = obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil // not really an error, but file doesn't exist
	} else if err != nil {
		return false, err
	}

	// Delete the File
	if err := obj.Delete(ctx); err != nil {
		return false, err
	}

	return true, nil
} // end func GcsDeleteFile

func GcsFileExists(bucketName string, objectName string) (bool, int64, time.Time, error) {
	updated := time.Time{}
	if bucketName == "" {
		bucketName = thisGcsStorageBucket // try the default
	}
	if bucketName == "" {
		return false, 0, updated, eer("No GCS Bucket provided")
	}
	if SubStr(objectName, 0, 1) == "/" {
		objectName = SubStr(objectName, 1)
	}
	if objectName == "" {
		return false, 0, updated, eer("No GCS Object Filename provided")
	}

	// Create the Storage Bucket Client
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return false, 0, updated, err
	}
	defer storageClient.Close()

	obj := storageClient.Bucket(bucketName).Object(objectName)

	// Check if the file exists
	attrs, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, 0, updated, nil
	} else if err != nil {
		return false, 0, updated, err
	}

	return true, attrs.Size, attrs.Updated, nil
} // end func GcsFileExists

func GcsFolderContentsOlderThan(bucketName, folderName string, threshold time.Duration) (contentsOlder, directoryExists bool, lastModifiedObjectName string, countThresholdFiles, countTotalFiles int64, err error) {
	contentsOlder = false
	directoryExists = false
	lastModifiedObjectName = ""
	countThresholdFiles = 0
	countTotalFiles = 0
	err = nil
	if bucketName == "" {
		bucketName = thisGcsStorageBucket // try the default
	}
	if bucketName == "" {
		err = eer("No GCS Bucket provided")
		return
	}
	if SubStr(folderName, 0, 1) == "/" {
		folderName = SubStr(folderName, 1)
	}
	if folderName == "" {
		err = eer("No GCS Folder provided")
		return
	}

	// Create the Storage Bucket Client
	ctx := context.Background()
	client, cerr := storage.NewClient(ctx)
	if cerr != nil {
		err = cerr
		return
	}
	defer client.Close()

	query := &storage.Query{Prefix: folderName}
	it := client.Bucket(bucketName).Objects(ctx, query)

	var latestTime time.Time
	for {
		attrs, aerr := it.Next()
		if aerr == iterator.Done {
			break
		}
		if aerr != nil {
			err = aerr
			return
		}

		countTotalFiles++
		directoryExists = true

		if attrs.Updated.After(latestTime) {
			latestTime = attrs.Updated
			lastModifiedObjectName = attrs.Name
		}

		if time.Since(attrs.Updated) <= threshold {
			countThresholdFiles++
		}
	} // end each file
	if directoryExists && latestTime != TimeZero() {
		contentsOlder = time.Since(latestTime) > threshold
	}
	err = nil
	return
} // end func GcsFolderContentsOlderThan

func GcsMoveFile(srcBucket string, srcObjectName string, dstBucket string, dstObjectName string) (bool, error) {
	start_move := time.Now()
	if SubStr(srcObjectName, 0, 1) == "/" {
		srcObjectName = SubStr(srcObjectName, 1)
	}
	if SubStr(dstObjectName, 0, 1) == "/" {
		dstObjectName = SubStr(dstObjectName, 1)
	}

	// Copy the Source File to Destination File
	copied, err := GcsCopyFile(srcBucket, srcObjectName, dstBucket, dstObjectName)
	if err != nil {
		return false, err
	} else if copied != true {
		return false, nil // File Not Found
	}

	// Delete the Source File
	_, err = GcsDeleteFile(srcBucket, srcObjectName)
	if err != nil {
		return true, err
	}
	o("File moved %s/%s to %s/%s [%s]", srcBucket, srcObjectName, dstBucket, dstObjectName, Runtime(start_move))
	return true, nil
} // end func GcsMoveFile

func GcsReadFile(bucketName string, objectName string, buf *bytes.Buffer) (bool, string, int64, time.Time, string, error) {
	var found bool = false
	var foundName string = objectName
	var mime string = ""
	var size int64 = 0
	var err error = nil
	updated := time.Time{}
	if bucketName == "" {
		bucketName = thisGcsStorageBucket // try the default
	}
	if bucketName == "" {
		return found, foundName, size, updated, mime, eer("No GCS Bucket provided")
	}
	if SubStr(objectName, 0, 1) == "/" {
		objectName = SubStr(objectName, 1)
	}
	if objectName == "" {
		return found, foundName, size, updated, mime, eer("No GCS Object Filename provided")
	}

	// Create the Storage API Client
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return found, foundName, size, updated, mime, err
	}
	defer storageClient.Close()

	// Connect to the Bucket
	bucket := storageClient.Bucket(bucketName)
	obj := bucket.Object(objectName)

	// Check if the File Exists
	attrs, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		err = nil // Not really an error, just not found
		now := time.Now()
		midnight := Today()
		nowd := now.Sub(midnight)
		if nowd < 2*time.Hour {
			today := now.Format("2006-01-02")
			if StrContains(objectName, "/"+today+"/") {
				yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
				testName := strings.ReplaceAll(objectName, "/"+today+"/", "/"+yesterday+"/")
				if testName != objectName {
					// Today file not found... Try Yesterday
					yfound, yname, ysize, yupdated, ymime, yerr := GcsReadFile(bucketName, testName, buf)
					if yfound && ysize > 0 && yerr == nil {
						return yfound, yname, ysize, yupdated, ymime, yerr
					} // end if yfound
				} // end if testName
			} // end if today
		} // end if nowd
		return found, foundName, size, updated, mime, err
	} else if err != nil {
		return found, foundName, size, updated, mime, err
	} // end if err

	// Read the File
	size = attrs.Size
	updated = attrs.Updated
	mime = attrs.ContentType
	found = true
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return found, foundName, size, updated, mime, err
	}
	defer reader.Close()

	// Copy to Buffer
	_, err = io.Copy(buf, reader)
	return found, foundName, size, updated, mime, err
} // end func GcsReadFile

func GcsReadFileBytes(bucketName string, objectName string) ([]byte, bool, string, int64, time.Time, string, error) {
	bytesData := make([]byte, 0)
	var found bool = false
	var foundName string = objectName
	var mime string = ""
	var size int64 = 0
	var err error = nil
	updated := time.Time{}
	if bucketName == "" {
		bucketName = thisGcsStorageBucket // try the default
	}
	if bucketName == "" {
		return bytesData, found, foundName, size, updated, mime, eer("No GCS Bucket provided")
	}
	if SubStr(objectName, 0, 1) == "/" {
		objectName = SubStr(objectName, 1)
	}
	if objectName == "" {
		return bytesData, found, foundName, size, updated, mime, eer("No GCS Object Filename provided")
	}

	// Read the file
	data := &bytes.Buffer{}
	found, foundName, size, updated, mime, err = GcsReadFile(bucketName, objectName, data)
	if !found || size == 0 || err != nil {
		return bytesData, found, foundName, size, updated, mime, err
	}
	bytesData, err = io.ReadAll(data)
	if err != nil {
		return bytesData, found, foundName, size, updated, mime, err
	}
	return bytesData, found, foundName, size, updated, mime, nil
} // end func GcsReadFileBytes

func GcsReadFileJSON[T any](bucketName string, objectName string, jsonObj *T) (bool, string, int64, time.Time, string, error) {
	bytesData, found, foundName, size, updated, mime, err := GcsReadFileBytes(bucketName, objectName)
	if found && size > 0 && err == nil {
		// JSON Decode
		err = JsonDecode(bytesData, &jsonObj)
	}
	return found, foundName, size, updated, mime, err
} // end func GcsReadFileJSON

func GcsSaveFile(bucketName string, objectName string, data any, contentType, contentDisposition string, metadata map[string]string) (int64, error) {
	if bucketName == "" {
		bucketName = thisGcsStorageBucket // try the default
	}
	if bucketName == "" {
		return 0, eer("No GCS Bucket provided")
	}
	if SubStr(objectName, 0, 1) == "/" {
		objectName = SubStr(objectName, 1)
	}
	if objectName == "" {
		return 0, eer("No GCS Object Filename provided")
	}

	// Convert the data (any type) to []byte
	dataBytes := AnyToByte(data)
	if len(dataBytes) == 0 {
		return 0, eer("No Data (0 bytes) for %s/%s", bucketName, objectName)
	}

	// Initialize the GCS Client
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return 0, eer("%+v", err)
	}
	defer storageClient.Close()

	// Initializat the GCS Object object
	bucket := storageClient.Bucket(bucketName)
	obj := bucket.Object(objectName)

	// Write the []byte data to file
	w := obj.NewWriter(ctx)
	if contentType != "" {
		w.ContentType = contentType
	}
	if len(metadata) > 0 {
		w.Metadata = metadata
	}
	if _, err := w.Write(dataBytes); err != nil {
		return 0, eer("%+v", err)
	}
	if err := w.Close(); err != nil {
		return 0, eer("%+v", err)
	}

	// Fetch the attributes of the object to get the file size
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return 0, eer("%+v", err)
	}

	return attrs.Size, nil
} // end func GcsSaveFile

///////////////////////////////////////////////////////////////////////////////////
//////// Text-to-Speech TTS Functions
///////////////////////////////////////////////////////////////////////////////////

func AddTtsPausesBytes(input []byte) []byte {
	anyFound := false
	output := input
	keys := DynamicPlaceholderBodyKeys(output) // map[ReplaceValue] = KeyName
	for replaceValue, keyName := range keys {
		keyPrefix := SubStr(keyName, 0, 6)
		if Lo(keyPrefix) == "pause_" && len(keyName) > 6 {
			// s(` <break time="%d" /> `, i) // this is GCP TTS specific
			var pause_s string = ""
			var pause_i int64 = 0
			key_x := StrSplit(keyName, "_")
			key_si := ""
			if len(key_x) > 0 {
				key_si = key_x[1]
			}
			if key_si != "" {
				pause_i = AnyToInt64(key_si)
				if pause_i > 0 {
					pause_s = s(` <break time="%d" /> `, pause_i)
				}
				o("Found pause key: %s: [@%s]: %d: ", keyName, replaceValue, pause_i, pause_s)
				output = bytes.ReplaceAll(output, []byte(replaceValue), []byte(pause_s))
				anyFound = true
			} // end if key_si
		} // end if pause_
	} // end foreach key
	if anyFound {
		o("input: %+v", string(input[:]))
		o("output: %+v", string(output[:]))
	}
	return output
} // end func AddTtsPausesBytes

func AddTtsPausesStr(input string) string {
	inBytes := []byte(input)
	outBytes := AddTtsPausesBytes(inBytes)
	return string(outBytes[:])
} // end func AddTtsPausesStr

func CheckTtsPausePlaceholders(tts *NewTtsRequest) (err error) {
	tts.Text = AddTtsPausesStr(tts.Text)
	return
} // end func CheckTtsPausePlaceholders

func ParseTtsGender(gender string) (texttospeechpb.SsmlVoiceGender, error) {
	switch gender {
	case "MALE", "male":
		return texttospeechpb.SsmlVoiceGender_MALE, nil
	case "FEMALE", "female":
		return texttospeechpb.SsmlVoiceGender_FEMALE, nil
	case "NEUTRAL", "neutral":
		return texttospeechpb.SsmlVoiceGender_NEUTRAL, nil
	default:
		return 0, eer("ParseTtsGender: Invalid gender: %s", gender)
	}
} // end func ParseTtsGender

func CheckTtsGender(g string) string {
	switch g {
	case "male", "female", "neutral":
		return strings.ToUpper(g)
	case "MALE", "FEMALE", "NEUTRAL":
		return g
	}
	return "FEMALE" // default to female
} // end func CheckTtsGender

func CheckTtsLanguage(l string) string {

	// TODO TODO -- Check against all voices

	return l
} // end func CheckTtsLanguage

// Creates the TTS File and Saves it to the Storage Bucket
func CreateNewTtsFile(ttsReq NewTtsRequest, filenames ...string) (string, int64, error) {
	// Check if we have a GCS Bucket Configured
	if thisTtsStorageBucket == "" {
		return "", 0, eer("CreateNewTtsFile: No GCS Bucket Configured")
	}

	if ttsReq.Text == "" {
		return "", 0, eer("CreateNewTtsFile: No Input Text Provided: %+v", ttsReq)
	}

	err := ValidateTtsRequest(&ttsReq)
	if err != nil {
		return "", 0, err
	}

	// Convert the request back to JSON to hash it
	jsonData, err := JsonEncode(ttsReq)
	if err != nil {
		return "", 0, err
	}
	if len(jsonData) == 0 {
		return "", 0, eer("CreateNewTtsFile: New TTS Request body was empty: %+v", ttsReq)
	}

	// Check the filename was included
	filename := ""
	if len(filenames) > 0 {
		if len(filenames[0]) > 0 {
			filename = filenames[0]
		}
	}
	if filename == "" {
		// No Name was provided, Create a SHA256 hash of the JSON data
		filename_hash := HashStr(jsonData)
		if filename_hash == "" {
			return "", 0, eer("CreateNewTtsFile: Error Hashing the Request to Filename")
		}
		filename = filename_hash + ".mp3"
	}
	save_file_path := GetTtsAudioFilePath(filename)
	if SubStr(save_file_path, 0, 1) == "/" {
		save_file_path = SubStr(save_file_path, 1) // We don't want the leading / for GCS
	}

	// Check if the files exists in the GCS Bucket
	exists, size, _, err := GcsFileExists(thisTtsStorageBucket, save_file_path)
	if err != nil {
		return "", 0, err
	} else if exists {
		o("CreateNewTtsFile: File found, using existing: %s", filename)
		return filename, size, nil
	}

	// Initialize the Text-to-Speech Client
	ctxTts := context.Background()
	ttsClient, err := texttospeech.NewClient(ctxTts)
	if err != nil {
		return "", 0, err
	}
	defer ttsClient.Close()

	// Create the TTS request using the text and language
	if ttsReq.Language == "" {
		ttsReq.Language = "en-US"
	}
	voiceSelection := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: ttsReq.Language,
	}

	// Set the voice type and gender if specified
	if ttsReq.Voice != "" {
		voiceSelection.Name = ttsReq.Voice
	}
	if ttsReq.Gender != "" {
		gender, err := ParseTtsGender(ttsReq.Gender)
		if err != nil {
			e("CreateNewTtsFile: Error parsing TTS Gender: %+v -- Using Neutral", err)
			gender = texttospeechpb.SsmlVoiceGender_NEUTRAL
		}
		voiceSelection.SsmlGender = gender
	}

	if Lo(SubStr(ttsReq.Text, 0, 7)) != "<speak>" {
		ttsReq.Text = "<speak>" + ttsReq.Text
	}
	if Lo(SubStr(ttsReq.Text, -8)) != "</speak>" {
		ttsReq.Text += "</speak>"
	}

	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Ssml{Ssml: ttsReq.Text},
		},
		Voice: voiceSelection,
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
			Pitch:         ttsReq.Pitch,
			SpeakingRate:  ttsReq.Speed,
			VolumeGainDb:  ttsReq.Volume,
		},
	}

	resp, err := ttsClient.SynthesizeSpeech(ctxTts, req)
	if err != nil {
		return "", 0, err
	}

	// Set the metadata
	contentType := "audio/mpeg"
	contentDisposition := "inline; filename=" + PathBasename(save_file_path)
	metadata := make(map[string]string)
	metadata["language"] = ttsReq.Language
	metadata["audio-format"] = "mp3"

	// Save the audio content to a file
	size, err = GcsSaveFile(thisTtsStorageBucket, save_file_path, resp.AudioContent, contentType, contentDisposition, metadata)
	if err != nil {
		return "", 0, err
	}

	o("CreateNewTtsFile: Audio file created: %s %d bytes", filename, size)
	return filename, size, nil
} // end func CreateNewTtsFile

// Get the Default TTS Voice for the Language and Gender
func DefaultTtsVoice(language, gender string) string {
	var voice string
	var key string
	var found bool
	var size int64
	var err error
	var ok bool
	key = language + "-" + gender
	voice, ok = thisTtsDefaultVoices[key].(string)
	if ok && voice != "" {
		return voice
	}

	// Get the GCP Voices List Cached API
	var defaultVoicesPath string
	fileTtsDefaultVoices := make(map[string]interface{})
	if thisGcsStorageBucket != "" {
		// Read the Defaults from GCS
		defaultVoicesPath = PathJoin(thisApiServicePrefix, "files", thisGcsBaseSavePath, "json", "configs", "tts", "defaultVoices.json")
		found, defaultVoicesPath, size, _, _, err = GcsReadFileJSON(thisGcsStorageBucket, defaultVoicesPath, &fileTtsDefaultVoices)
		if found && size > 0 && err == nil {
			thisTtsDefaultVoices, _ = MergeMaps(fileTtsDefaultVoices, thisTtsDefaultVoices)
			voice, ok = thisTtsDefaultVoices[key].(string)
			if ok && voice != "" {
				return voice
			}
		}
	} // end if thisGcsStorageBucket

	// Get all voices
	vl := GetAllTtsVoicesList()
	//o("All TTS Voices List count: %d", len(vl.Voices))

	ttsVoice, found := vl.Voices[voice]
	if found {
		o("Voice Matched: %s", ttsVoice.Name)
	}

	// Set the Global Default
	defaultVoicesChanged := false
	thisKeyValue, ok := thisTtsDefaultVoices[key].(string)
	if thisKeyValue != voice {
		thisTtsDefaultVoices[key] = voice
		defaultVoicesChanged = true
	}

	if defaultVoicesChanged && thisGcsStorageBucket != "" && defaultVoicesPath != "" {
		// Set the metadata
		contentType := "application/json"
		contentDisposition := "inline; filename=" + PathBasename(defaultVoicesPath)
		metadata := make(map[string]string)
		metadata["language"] = "en-US"

		// Save the Defaults to GCS
		size, err := GcsSaveFile(thisGcsStorageBucket, defaultVoicesPath, thisTtsDefaultVoices, contentType, contentDisposition, metadata)
		if size == 0 || err != nil {
			e("Failed to Save the TtsDefaultVoices: %+v", thisTtsDefaultVoices)
		}
	} // end if thisGcsStorageBucket
	return voice
} // end func GetDefaultTtsVoice

func GetAllTtsVoicesListGcp() (MapTtsVoicesList, error) {
	var vl MapTtsVoicesList
	var err error
	vl.Voices = make(map[string]TtsVoice)
	ctx := context.Background()
	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return vl, err
	}
	defer client.Close()

	// Performs the list voices request.
	resp, err := client.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return vl, err
	}

	for _, voice := range resp.Voices {
		var v TtsVoice

		v.Name = voice.Name
		v.LanguageCodes = voice.LanguageCodes
		v.SsmlGender = voice.SsmlGender.String()
		v.SampleRateHz = voice.NaturalSampleRateHertz

		vl.Voices[v.Name] = v
	}

	return vl, nil
} // end func GetAllTtsVoicesListGcp

func GetAllTtsVoicesList() MapTtsVoicesList {
	var fileVoices MapTtsVoicesList
	fileVoices.Voices = make(map[string]TtsVoice)
	anyChanges := false

	// Check if we loaded it in this execution
	if len(thisTtsAllVoicesList.Voices) > 0 {
		return thisTtsAllVoicesList
	}

	// Get the GCP Voices List Cached API
	voicesListPath := PathJoin(thisApiServicePrefix, "files", thisGcsBaseSavePath, "json", "configs", "tts", "allTtsVoices.json")
	found, voicesListPath, size, updated, _, err := GcsReadFileJSON(thisGcsStorageBucket, voicesListPath, &fileVoices)
	if !found || size == 0 || err != nil || TimeSince(updated) > TimeDuration(86400) {
		// GCS File doesn't exist or is not readable or is older than 1 day

		// Get the list from the API
		////// GET https://texttospeech.googleapis.com/v1/voices        // Optional: ?languageCode={BCP-47 Language Code: "en" or "en-US"}
		apiList, err := GetAllTtsVoicesListGcp()
		if err == nil && len(apiList.Voices) > 0 {
			for n, v := range apiList.Voices {
				fileVoices.Voices[n] = v
				anyChanges = true
			} // end foreach TtsVoice
		}

		if anyChanges {
			// Set the metadata
			contentType := "application/json"
			contentDisposition := "inline; filename=" + PathBasename(voicesListPath)
			metadata := make(map[string]string)
			metadata["language"] = "en-US"

			// Save the updated list to the GCS file
			size, err := GcsSaveFile(thisGcsStorageBucket, voicesListPath, fileVoices, contentType, contentDisposition, metadata)
			if size == 0 || err != nil {
				e("GetAllTtsVoicesList: Error Saving the MapTtsVoicesList: %+v", fileVoices)
			}
		} // end if ok
	} // end if GCS file

	// Set the global variable
	thisTtsAllVoicesList = fileVoices
	return fileVoices
} // end func GetAllTtsVoicesList

// Get the Default TTS Request
func GetNewTtsRequest(voice, language, gender string, pitch, speed, volume float64) NewTtsRequest {
	var t NewTtsRequest
	// Defaults
	//Gender      string  `json:"gender,omitempty"`   // e.g., "MALE", "FEMALE", "NEUTRAL"
	//Pitch       float64 `json:"pitch,omitempty"`    // Range: -20.0 to 20.0, Default 0.00
	//Speed       float64 `json:"speed,omitempty"`    // Range: 0.25 to 4.0, Default 1.00
	//Volume      float64 `json:"volume,omitempty"`   // Range: -96.0 to 16.0, Default 0.00
	t.Text = ""

	language = CheckTtsLanguage(language)
	t.Language = language
	gender = CheckTtsGender(gender)
	t.Gender = gender

	if voice != "" {
		vl := GetAllTtsVoicesList()
		if _, ok := vl.Voices[voice]; ok {
			t.Voice = voice
		}
	}
	if t.Voice == "" {
		t.Voice = DefaultTtsVoice(language, gender)
	}
	t.Pitch = pitch
	t.Speed = speed
	t.Volume = volume
	if t.Pitch < -20 {
		t.Pitch = -20
	} else if t.Pitch > 20 {
		t.Pitch = 20
	}
	if t.Speed < 0.25 {
		t.Speed = 0.25
	} else if t.Speed > 4 {
		t.Speed = 4
	}
	if t.Volume < -96 {
		t.Volume = -96
	} else if t.Volume > 16 {
		t.Volume = 16
	}
	return t
} // end func GetNewTtsRequest

func GetTtsAudioFileFromPath(request_path string) string {
	audio_file := GetRequestIdFromPath(request_path, thisTtsAudioSavePath, "audio", thisApiServicePrefix)
	return audio_file
} // end func GetTtsAudioFileFromPath

func GetTtsAudioFilePath(audio_file string) string {
	audio_file_path := PathJoin(thisApiServicePrefix, "audio", thisTtsAudioSavePath, audio_file)
	return audio_file_path
} // end func GetTtsAudioFilePath

func GetTtsAudioFileUrl(request_base_url string, audio_file string) string {
	audio_file_url := PathJoin(request_base_url, thisApiServicePrefix, "audio", thisTtsAudioSavePath, audio_file)
	return audio_file_url
} // end func GetTtsAudioFileUrl

// Get the Audio URL for a new TTS API Request
func TtsCreateNewAudio(r *http.Request, tts NewTtsRequest) (resp NewTtsResponse, err error) {
	err = nil
	resp.Success = false
	if tts.Text == "" {
		err = eer("TtsCreateNewAudio: TTS Text Cannot be Empty")
		return
	}
	tts_start := time.Now()
	ttsUrl := GetTtsApiUrl(r)
	ttsKey := GetTtsApiKey()
	headers := make(map[string]string)
	headers["x-api-key"] = ttsKey
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	d("TtsCreateNewAudio: TTS Url: %s", ttsUrl)
	d("TtsCreateNewAudio: TTS Request: %+v", tts)
	rp, err := NewRestApiRequest(http.MethodPost, ttsUrl, &resp, headers, tts, 60, 0)
	if err != nil {
		return
	}
	if rp.GetCode() == 0 {
		rp.SetCode(500)
	}
	//o("TTS API Result [%d] [%s]: %+v", rp.GetCode(), Runtime(tts_start), result)
	if rp.GetCode() != 200 && rp.GetCode() != 202 {
		o("TtsCreateNewAudio: TTS API Result [%d] [%s]: %+v", rp.GetCode(), Runtime(tts_start), rp)
		resp.Success = false
		err = eer("TtsCreateNewAudio: TTS API Result Error [%d]", rp.GetCode())
		return
	}
	err = nil
	return
} // end func TtsCreateNewAudio

func ValidateTtsRequest(ttsReq *NewTtsRequest) error {
	// Create the TTS request using the text and language
	if ttsReq.Language == "" {
		ttsReq.Language = "en-US"
	}

	// Check the Float input ranges
	// Set default for Speed should be 1.00 if 0.00
	if ttsReq.Speed == 0 {
		ttsReq.Speed = 1.0
	} else if ttsReq.Speed < 0.25 {
		ttsReq.Speed = 0.25
	} else if ttsReq.Speed > 4.0 {
		ttsReq.Speed = 4.0
	}
	if ttsReq.Pitch < -20.0 {
		ttsReq.Pitch = -20.0
	} else if ttsReq.Pitch > 20.0 {
		ttsReq.Pitch = 20.0
	}
	if ttsReq.Volume < -96.0 {
		ttsReq.Volume = -96.0
	} else if ttsReq.Volume > 16.0 {
		ttsReq.Volume = 16.0
	}

	if ttsReq.Gender != "" {
		_, err := ParseTtsGender(ttsReq.Gender)
		if err != nil {
			e("Failed to Parse TTS Gender: %+v", err)
			ttsReq.Gender = ""
		}
	}
	if ttsReq.Gender == "" {
		o("TTS Gender: Not Set or Invalid -- Using Neutral")
		ttsReq.Gender = "NEUTRAL"
	}

	return nil
} // end func ValidateTtsRequest

func GetLocalIPs() []string {
	ips := make([]string, 0)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//return ipnet.IP.String()
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
} // end func GetLocalIPs
