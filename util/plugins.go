package util

var (
	GOTM_LOG_AREA = NewArea("GOTM.Log")
)

// side effect executes request.ParseForm
func log(w HttpWriter, r HttpReq, info map[string]any) any {
	r.ParseForm()
	FLog(GOTM_LOG_AREA, "new HTTP conn [%s] @ %s {%+v}\n", r.Method, r.URL.Path, r.Form)
	return nil
}
var GOTM_Log = GOTMPlugin{"_logging", log}
