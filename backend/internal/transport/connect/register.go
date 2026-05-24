package connect
import (
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/connect"
	"net/http"
)
func Register(mux *http.ServeMux, svc *connect.MtHubServer) {
	antv1c.NewMtHubServiceHandler(svc, mux)
}
