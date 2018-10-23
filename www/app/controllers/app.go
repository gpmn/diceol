package controllers

import (
	"log"

	"github.com/revel/revel"
)

// CommResp :
type CommResp struct {
	Result int
	Desc   string
	Data   interface{}
}

// App :
type App struct {
	*revel.Controller
}

// Index :
func (c App) Index() revel.Result {
	return c.Redirect(App.EosForce)
}

// GetBetHistory :
func (c App) GetBetHistory(chain, player string, highBound uint64, limit int) revel.Result {
	hises, err := rsvCtrl.GetDiceHisTbl(player, highBound, limit)
	if nil != err {
		log.Printf("App.GetBetHistory - failed : %v")
		return c.RenderJSON(CommResp{
			Result: -1,
			Desc:   "DB query error",
			Data:   nil,
		})
	}
	// if len(hises) > 0 {
	// 	log.Printf("App.GetBetHistory - from %d_%d -> %d_%d",
	// 		hises[0].AccountActSeq,
	// 		hises[0].OsID,
	// 		hises[len(hises)-1].AccountActSeq,
	// 		hises[len(hises)-1].OsID)
	// }
	return c.RenderJSON(CommResp{
		Result: 0,
		Desc:   "OK",
		Data:   hises,
	})
}

// GetGrpHisTbl :
func (c App) GetGrpHisTbl(chain, group string, limit, offset int64) revel.Result {
	grps, err := rsvCtrl.GetGrpHisTbl(group, limit, offset)
	if nil != err {
		return c.RenderJSON(CommResp{
			Result: -1,
			Desc:   "DB query error",
			Data:   nil,
		})
	}

	return c.RenderJSON(CommResp{
		Result: 0,
		Desc:   "OK",
		Data:   grps,
	})
}

// EosForce :
func (c App) EosForce() revel.Result {
	return c.Render()
}

func (c App) Eos() revel.Result {
	return c.Render()
}
