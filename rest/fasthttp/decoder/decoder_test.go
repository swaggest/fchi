package decoder_test

//type failingValidator struct{}
//
//var _ request.Validator = failingValidator{}
//
//func (failingValidator) ValidateRequestData(in request.In, namedData map[string]interface{}) error {
//	return errors.New("failed again")
//}
//
//const badRequestResponse = `{"status":"INVALID_ARGUMENT","error":"failed again"}`
//
//func TestHandlerWithRequestMapper_ServeHTTP(t *testing.T) {
//	rd := request.HandlerWithRequestMapper{
//		Handler:   makeHandler(t),
//		Mapper:    form.NewMapperFactory().MakeMapper(http.MethodPost, makeHandler(t).Request),
//		Validator: nil,
//	}
//	rd.SetRequest(makeHandler(t).Request)
//
//	w := httptest.NewRecorder()
//	r := httptest.NewRequest(http.MethodPost, "/here/30?name=John",
//		bytes.NewReader([]byte(`{"meta":["one","two","three"]}`)))
//
//	router := chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	expected := `{"whatever":12}`
//	response := strings.Trim(w.Body.String(), "\n")
//	assert.Equal(t, expected, response)
//
//	w = httptest.NewRecorder()
//	r = httptest.NewRequest(http.MethodPost, "/here/notInt?name=John",
//		bytes.NewReader([]byte(`{"meta":["one","two","three"]}`)))
//
//	router = chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	expected = `{
//		"status":"INVALID_ARGUMENT",
//		"error":"request decoding failed",
//		"context":{"errors":{"path:age":["#: Invalid Integer Value 'notInt' Type 'int' Namespace 'age'"]}}
//	}`
//	assertjson.Equals(t, []byte(expected), w.Body.Bytes())
//
//	w = httptest.NewRecorder()
//	r = httptest.NewRequest(http.MethodPost, "/here/30?name=John", bytes.NewReader([]byte(`{"meta":[1,2,3]}`)))
//
//	router = chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	assert.Equal(t, http.StatusBadRequest, w.Code)
//
//	w = httptest.NewRecorder()
//	r = httptest.NewRequest(http.MethodPost, "/here/30?name=John&is=abc", bytes.NewReader([]byte(`{"meta":[1,2,3]}`)))
//
//	router = chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	expected = `{
//		"status":"INVALID_ARGUMENT",
//		"error":"request decoding failed",
//		"context":{"errors":{"query:is":["#: Invalid Boolean Value 'abc' Type 'bool' Namespace 'is'"]}}
//	}`
//	jsondiff.AssertJSONEquals(t, []byte(expected), w.Body.Bytes())
//}
//
//func TestHandlerWithRequestMapper_ServeHTTPFailing(t *testing.T) {
//	rd := request.HandlerWithRequestMapper{
//		Handler:   makeHandler(t),
//		Mapper:    form.NewMapperFactory().MakeMapper(http.MethodPost, makeHandler(t).Request),
//		Validator: failingValidator{},
//	}
//	rd.SetRequest(makeHandler(t).Request)
//
//	w := httptest.NewRecorder()
//	r := httptest.NewRequest(http.MethodPost, "/here/30?name=John",
//		bytes.NewReader([]byte(`{"meta":["one","two","three"]}`)))
//
//	router := chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	expected := badRequestResponse
//	response := strings.Trim(w.Body.String(), "\n")
//	assert.Equal(t, expected, response)
//
//	w = httptest.NewRecorder()
//	r = httptest.NewRequest(http.MethodPost, "/here/123?name=John",
//		bytes.NewReader([]byte(`{"meta":["one","two","three"]}`)))
//
//	router = chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	expected = badRequestResponse
//	response = strings.Trim(w.Body.String(), "\n")
//	assert.Equal(t, expected, response)
//
//	w = httptest.NewRecorder()
//	r = httptest.NewRequest(http.MethodPost, "/here/30?name=John", bytes.NewReader([]byte(`{"meta":[1,2,3]}`)))
//
//	router = chi.NewRouter()
//	router.Method(http.MethodPost, "/here/{age}", &rd)
//
//	router.ServeHTTP(w, r)
//
//	expected = badRequestResponse
//	response = strings.Trim(w.Body.String(), "\n")
//	assert.Equal(t, expected, response)
//}
//
//func TestMapper_DecodeWithContentType(t *testing.T) {
//	type req struct {
//		Val int `json:"val"`
//	}
//
//	r := httptest.NewRequest(http.MethodPost, "/here/notInt?name=John", bytes.NewReader([]byte(`{"val":1}`)))
//	r.Header.Set("content-type", "application/json;charset=UTF-8")
//
//	m := form.NewMapperFactory().MakeMapper(http.MethodPost, new(req))
//	v := req{}
//	err := m.Decode(r, &v, nil)
//	assert.NoError(t, err)
//	assert.Equal(t, 1, v.Val)
//
//	r = httptest.NewRequest(http.MethodPost, "/here/notInt?name=John", bytes.NewReader([]byte(`a,b,c`)))
//	r.Header.Set("content-type", "text/csv")
//
//	v = req{}
//	err = m.Decode(r, &v, nil)
//	assert.EqualError(t, err, "request with 'application/json' content type expected")
//	assert.Equal(t, 0, v.Val, "value should not change")
//
//	r = httptest.NewRequest(http.MethodPost, "/here/notInt?name=John", bytes.NewReader([]byte(`{"val":2}`)))
//	r.Header.Set("content-type", "application/json")
//
//	v = req{}
//	err = m.Decode(r, &v, nil)
//	assert.NoError(t, err)
//	assert.Equal(t, 2, v.Val)
//}
//
//func TestMapper_DecodeArrayBody(t *testing.T) {
//	type req []struct {
//		Val int `json:"val"`
//	}
//
//	r := httptest.NewRequest(http.MethodPost, "/here/notInt?name=John", bytes.NewReader([]byte(`[{"val":1},{"val":2}]`)))
//	r.Header.Set("content-type", "application/json;charset=UTF-8")
//
//	m := form.NewMapperFactory().MakeMapper(http.MethodPost, new(req))
//	v := req{}
//	err := m.Decode(r, &v, nil)
//	assert.NoError(t, err)
//	assert.Equal(t, 1, v[0].Val)
//	assert.Equal(t, 2, v[1].Val)
//}
//
//func TestMapper_DecodeMapBody(t *testing.T) {
//	type req map[string]struct {
//		Val int `json:"val"`
//	}
//
//	r := httptest.NewRequest(http.MethodPost, "/here/notInt?name=John",
//		bytes.NewReader([]byte(`{"key1":{"val":1},"key2":{"val":2}}`)))
//	r.Header.Set("content-type", "application/json;charset=UTF-8")
//
//	m := form.NewMapperFactory().MakeMapper(http.MethodPost, new(req))
//	v := req{}
//	err := m.Decode(r, &v, nil)
//	assert.NoError(t, err)
//	assert.Equal(t, 1, v["key1"].Val)
//	assert.Equal(t, 2, v["key2"].Val)
//}
//
//func TestMapper_DecodeCustom(t *testing.T) {
//	type constName string
//
//	type req struct {
//		Goal constName `query:"name"`
//	}
//
//	f := form.NewMapperFactory()
//	f.RegisterFunc(func(s string) (interface{}, error) {
//		if s != "Constantine" {
//			return nil, errors.New(`expected "Constantine" value`)
//		}
//		return constName(s), nil
//	}, constName(""))
//
//	m := f.MakeMapper(http.MethodGet, new(req))
//	v := req{}
//	err := m.Decode(httptest.NewRequest(http.MethodGet, "/here/notInt?name=John", nil), &v, nil)
//	assert.EqualError(t, err, "request decoding failed")
//
//	err = m.Decode(httptest.NewRequest(http.MethodGet, "/here/notInt?name=Constantine", nil), &v, nil)
//	assert.NoError(t, err)
//	assert.Equal(t, constName("Constantine"), v.Goal)
//}
//
//type embeddedMeta struct {
//	Meta []string `json:"meta,omitempty"`
//}
//
//func makeHandler(t *testing.T) *rest.Handler {
//	type handlerRequest struct {
//		Age  int    `path:"age"`
//		Goal string `query:"name"`
//		Is   *bool  `query:"is" required:"false"`
//		embeddedMeta
//	}
//
//	type handlerResponse struct {
//		Whatever int `json:"whatever"`
//	}
//
//	u := usecase.FunctionalInteractor{}
//	u.Title = "My Handler"
//	u.Description = "It handles"
//	u.Input = new(handlerRequest)
//	u.Output = new(handlerResponse)
//	u.InteractFunc = func(ctx context.Context, input, output interface{}) error {
//		r, ok := input.(*handlerRequest)
//		assert.True(t, ok)
//		o, ok := output.(*handlerResponse)
//		assert.True(t, ok)
//
//		if r.Age != 30 {
//			t.Fatal("wrong age parameter value")
//		}
//
//		if r.Goal != "John" {
//			t.Fatal("wrong name parameter value")
//		}
//
//		if strings.Join(r.Meta, ",") != "one,two,three" {
//			t.Fatal("wrong meta parameter value")
//		}
//
//		o.Whatever = 12
//
//		return nil
//	}
//
//	h := rest.NewUseCaseHandler(u)
//
//	return h
//}
//
//func TestFuzz(t *testing.T) {
//	type prop1 struct {
//		Deeper int `json:"deeper"`
//	}
//
//	type input struct {
//		Cookie float64 `cookie:"cookie"`
//		Query  int     `query:"query"`
//		Path   string  `path:"path"`
//		Header bool    `header:"Header"`
//		Prop1  prop1   `json:"prop1"`
//		Prop2  string  `json:"prop2"`
//	}
//
//	m := form.NewMapperFactory().MakeMapper(http.MethodPost, new(input))
//
//	req, err := http.NewRequest(http.MethodPost, "/some/url?query=123&notRelevant=bar",
//		bytes.NewBufferString(`{"prop1":{"deeper": 100}, "prop2":"hello", "prop3": 3.14}`))
//	require.NoError(t, err)
//	req.Header.Set("Header", "true")
//	req.Header.Set("Not-Relevant", "3.15")
//	req.Header.Set("Content-Length", "10000")
//	req.AddCookie(&http.Cookie{
//		Goal:  "cookie",
//		Value: "3.14",
//	})
//	req.AddCookie(&http.Cookie{
//		Goal:  "notRelevant",
//		Value: "abc",
//	})
//
//	dump, err := httputil.DumpRequest(req, true)
//	require.NoError(t, err)
//	require.NoError(t, ioutil.WriteFile("./fuzz/corpus/req.txt", dump, 0640))
//
//	data, err := ioutil.ReadFile("./fuzz/corpus/req.txt")
//	require.NoError(t, err)
//
//	req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(data)))
//	require.NoError(t, err)
//
//	r := chi.NewRouter()
//	r.Method(http.MethodPost, "/some/{path}", http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
//		i := input{}
//		err = m.Decode(req, &i, nil)
//		require.NoError(t, err)
//	}))
//
//	r.ServeHTTP(nil, req)
//}
