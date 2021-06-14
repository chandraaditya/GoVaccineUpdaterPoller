package webhook

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUpdateDistricts(t *testing.T) {
	viper.SetConfigType("yaml")
	var yamlExample = []byte(`
api-keys:
  my-test-project-key:
    slot-open-webhook: "http://localhost:8000/open_hook"
    slot-closed-webhook: "http://localhost:8000/close_hook"
    districts:
      - 294
      - 10
`)
	tmpfile, err := ioutil.TempFile("", "example.*.yaml")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(yamlExample); err != nil {
		_ = tmpfile.Close()
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}
	viper.SetConfigFile(tmpfile.Name())
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	districts, err := NewDistricts()
	require.NoError(t, err)
	require.Equal(t, len(districts.GetDistricts()), 2)

	requestJson := `{
	"slot_open_webhook": "https://google.com",
	"slot_closed_webhook": "https://yahoo.com",
	"districts": [1,2,3]
}`
	request := httptest.NewRequest("POST", "/update_district", strings.NewReader(requestJson))
	request.Header.Add("X-Api-Key", "my-test-project-key")
	responseRecorder := httptest.NewRecorder()

	handler := &server{}
	handler.UpdateDistricts(responseRecorder, request)

	contents, _ := ioutil.ReadFile(tmpfile.Name())
	expected := `api-keys:
  my-test-project-key:
    districts:
    - 1
    - 2
    - 3
    slot-closed-webhook: https://yahoo.com
    slot-open-webhook: https://google.com
`
	println(string(contents))
	require.Equal(t, expected, string(contents))
	require.Equal(t, http.StatusOK, responseRecorder.Code)
}

func TestUpdateDistrictsNegativeCases(t *testing.T) {
	tt := []struct {
		name       string
		method     string
		body       string
		want       string
		statusCode int
		key        string
	}{
		{
			name:       "get method",
			method:     http.MethodGet,
			want:       "{\"message\": \"this endpoint only supports post http requests contact the developer for more information\"}",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "bad data",
			method:     http.MethodPost,
			body:       "abdfd",
			want:       "{\"message\": \"bad request\"}",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "no key",
			method:     http.MethodPost,
			body:       "{}",
			want:       "{\"message\": \"unauthorized request\"}",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "wrong key",
			method:     http.MethodPost,
			key:        "wrong key",
			body:       "{}",
			want:       "{\"message\": \"unauthorized request\"}",
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			config = &APIKeys{ApiKeys: map[string]Config{
				"api-key-1": Config{},
			}}
			request := httptest.NewRequest(tc.method, "/update_district", strings.NewReader(tc.body))
			if tc.key != "" {
				request.Header.Add("X-Api-Key", tc.key)
			}
			responseRecorder := httptest.NewRecorder()

			handler := &server{}
			handler.UpdateDistricts(responseRecorder, request)

			if responseRecorder.Code != tc.statusCode {
				t.Errorf("Want status '%d', got '%d'", tc.statusCode, responseRecorder.Code)
			}

			if strings.TrimSpace(responseRecorder.Body.String()) != tc.want {
				t.Errorf("Want '%s', got '%s'", tc.want, responseRecorder.Body)
			}
		})
	}
}
