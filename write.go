package fp

import (
	"encoding/json"
	"net/http"
)

// ContentType is the media type of a Problem Details document (RFC 9457 §3).
const ContentType = "application/problem+json"

// Write writes p as an HTTP response: the Content-Type header set to
// [ContentType], p.Status as the status code, and p as the JSON body.
//
// p is marshaled before anything is written, so a marshal error is returned
// with the response untouched and the caller can still answer some other way.
// An error after that comes from the ResponseWriter.
//
//	if err != nil {
//	    _ = fp.Write(w, fp.FromError(err, myDomainMapper))
//	    return
//	}
func Write(w http.ResponseWriter, p *Problem) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", ContentType)
	w.WriteHeader(p.Status)
	_, err = w.Write(append(b, '\n'))
	return err
}
