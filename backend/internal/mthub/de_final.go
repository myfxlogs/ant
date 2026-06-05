package main
import ("context";"fmt";"time";"github.com/google/uuid";"github.com/jackc/pgx/v5/pgxpool";"anttrader/internal/repository";"google.golang.org/protobuf/proto";antv1 "anttrader/gen/proto/ant/v1")
func main() {
ctx:=context.Background()
pool,_:=pgxpool.New(ctx,"postgres://ant:QxhrPqrizFg0iTWNOnabaFvv@localhost:5433/ant?sslmode=disable")
defer pool.Close()
repo:=repository.NewStrategyExperimentRepository(pool)
uid:=uuid.MustParse("3b7a8c21-4d84-42a5-b550-57c9867ec333")
code:="# @param fast_period 10 range=5:50:5\n# @param slow_period 30 range=10:100:10\ndef run(context):\n    p = context.get('params', {})\n    fast_period = int(p.get('fast_period', 10))\n    slow_period = int(p.get('slow_period', 30))\n    prices = context['close']\n    if len(prices) < slow_period + 1:\n        return {'signal': 'hold', 'volume': 0}\n    fast_ma = sum(prices[-fast_period:]) / fast_period\n    slow_ma = sum(prices[-slow_period:]) / slow_period\n    pos = context.get('position')\n    if fast_ma > slow_ma:\n        if pos and pos.get('side') == 'buy':\n            return {'signal': 'hold', 'volume': 0}\n        if pos:\n            return {'signal': 'close', 'volume': 0}\n        return {'signal': 'buy', 'volume': 1.0}\n    elif fast_ma < slow_ma:\n        if pos and pos.get('side') == 'sell':\n            return {'signal': 'hold', 'volume': 0}\n        if pos:\n            return {'signal': 'close', 'volume': 0}\n        return {'signal': 'sell', 'volume': 1.0}\n    return {'signal': 'hold', 'volume': 0}\n"
exp:=&repository.StrategyExperiment{ID:uuid.New(),UserID:uid,StrategyCode:code,SearchMethod:"de",MaxCandidates:10}
repo.Create(ctx,exp)
fmt.Printf("Exp: %s\n",exp.ID)
for i:=0;i<120;i++{time.Sleep(5*time.Second);var s string;pool.QueryRow(ctx,"SELECT status FROM strategy_experiments WHERE id=$1",exp.ID).Scan(&s)
if s=="COMPLETED"||s=="FAILED"{fmt.Printf("Status: %s\n",s)
if s=="FAILED"{fmt.Println("FAIL");return}
rows,_:=pool.Query(ctx,"SELECT rank,score,grade,parameters FROM strategy_experiment_candidates WHERE experiment_id=$1 ORDER BY rank",exp.ID)
cnt:=0;prev:=-1.0;same:=true
for rows.Next(){var r int;var sc float64;var g string;var p []byte;rows.Scan(&r,&sc,&g,&p)
var cp antv1.CandidateParameters;proto.Unmarshal(p,&cp)
fmt.Printf("  #%d: score=%.1f grade=%s params=%v\n",r,sc,g,cp.GetValues())
if prev>0&&sc!=prev{same=false};prev=sc;cnt++}
if cnt>0&&!same{fmt.Println("PASS")}else{fmt.Println("FAIL: all same")}
return}}
if i%12==0{fmt.Printf("  %ds (%s)\n",i*5,s)}}
fmt.Println("FAIL: timeout")}
