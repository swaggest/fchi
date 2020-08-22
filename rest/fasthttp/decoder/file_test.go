package decoder_test

import (
	"testing"
)

func TestMapper_Decode_fileUploadTag(t *testing.T) {
	//type ReqEmb struct {
	//	UploadHeader *multipart.FileHeader `formData:"upload"`
	//}
	//
	//type req struct {
	//	ReqEmb
	//	Upload multipart.File `file:"upload"`
	//}
	//
	//r := chi2.NewWrapper(
	//	chi.NewRouter(),
	//	(&swagger.Swaggerizer{}).HandlerWare,
	//	request.NewRequestMapperHandlerWare(form.NewMapperFactory()),
	//)
	//
	//u := usecase.FunctionalInteractor{}
	//u.Input = new(req)
	//u.InteractFunc = func(ctx context.Context, input, output interface{}) error {
	//	inp, ok := input.(*req)
	//	assert.True(t, ok)
	//	assert.NotNil(t, inp.Upload)
	//	assert.NotNil(t, inp.UploadHeader)
	//	assert.Equal(t, "my.csv", inp.UploadHeader.Filename)
	//	assert.Equal(t, int64(6), inp.UploadHeader.Size)
	//	content, err := ioutil.ReadAll(inp.Upload)
	//	assert.NoError(t, err)
	//	assert.Equal(t, "Hello!", string(content))
	//
	//	return nil
	//}
	//
	//h := rest.NewUseCaseHandler(u, func(h *rest.Handler) {
	//	h.Path = "/receive"
	//	h.Method = http.MethodPost
	//})
	//
	//router.AddHandlers(r, h)
	//
	//srv := httptest.NewServer(r)
	//defer srv.Close()
	//
	//var b bytes.Buffer
	//w := multipart.NewWriter(&b)
	//
	//writer, err := w.CreateFormFile("upload", "my.csv")
	//assert.NoError(t, err)
	//
	//_, err = writer.Write([]byte(`Hello!`))
	//assert.NoError(t, err)
	//
	//assert.NoError(t, w.Close())
	//
	//hreq := httptest.NewRequest(http.MethodPost, srv.URL+"/receive", &b)
	//hreq.RequestURI = ""
	//hreq.Header.Set("Content-Type", w.FormDataContentType())
	//
	//resp, err := srv.Client().Do(hreq)
	//assert.NoError(t, err)
	//assert.NoError(t, resp.Body.Close())
}
