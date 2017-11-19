package tactics

import (
	f "framework"
	"math/rand"
)

type Less struct {
	prevTarget map[string]*f.Position
}

func NewLess() *Less {
	inst := &Less {
		prevTarget: make(map[string]*f.Position),
	}
	return inst
}

func (self *Less) Init(state *f.GameState) {
}

func (self *Less) Plan(state *f.GameState, radar *f.RadarResult, objective map[string]f.Objective) {
	fireForest := make(map[f.Position]FireForest)
	tankloop: for _, tank := range state.MyTank {
		fireForest[f.Position { X: tank.Pos.X - 1, Y: tank.Pos.Y }] = FireForest { tank.Id, f.ActionFireLeft }
		fireForest[f.Position { X: tank.Pos.X + 1, Y: tank.Pos.Y }] = FireForest { tank.Id, f.ActionFireRight }
		fireForest[f.Position { X: tank.Pos.X, Y: tank.Pos.Y - 1 }] = FireForest { tank.Id, f.ActionFireUp }
		fireForest[f.Position { X: tank.Pos.X, Y: tank.Pos.Y + 1}] = FireForest { tank.Id, f.ActionFireDown }
		fireRadar := radar.Fire[tank.Id]
		for _, fire := range []*f.RadarFire { fireRadar.Up, fireRadar.Down, fireRadar.Left, fireRadar.Right } {
			if fire != nil && fire.Sin < 0.2 && fire.Faith > 0.2 {
				objective[tank.Id] = f.Objective {
					Action: fire.Action,
				}
				continue tankloop
			}
		}

		least := 99999
		var ttank *f.Tank
		for _, etank := range state.EnemyTank {
			dist := abs(tank.Pos.X - etank.Pos.X) + abs(tank.Pos.Y - etank.Pos.Y)
			if dist < least {
				ttank = &etank
				least = dist
			}
		}
		pos := f.Position {}
		if ttank != nil {
			pos = ttank.Pos
			// 5,8 - 10,12
			if pos.X < 5 {
				pos.X = 5
			}
			if pos.X > 10 {
				pos.X = 10
			}
			if pos.Y < 8 {
				pos.Y = 8
			}
			if pos.Y > 12 {
				pos.Y = 12
			}
		} else {
			target := self.prevTarget[tank.Id]
			if target == nil || target.SDist(tank.Pos) < state.Params.TankSpeed || rand.Int() % 8 == 0 {
				target = &f.Position {
					X: rand.Int() % (10 - 5) + 5,
					Y: rand.Int() % (12 - 8) + 8,
				}
				self.prevTarget[tank.Id] = target
			}
			pos = *target
			if tank.Pos.X > 4 && state.Terain.Get(tank.Pos.X, tank.Pos.Y) == 2 && rand.Int() % 4 == 0 {
				fires := []*f.RadarFire { fireRadar.Up, fireRadar.Down, fireRadar.Left, fireRadar.Right }
				for _, i := range rand.Perm(4) {
					fire := fires[i]
					if fire != nil && fire.Sin < 0.1 {
						objective[tank.Id] = f.Objective {
							Action: fire.Action,
						}
						continue tankloop
					}
				}
			}
		}
		objective[tank.Id] = f.Objective {
			Action: f.ActionTravelWithDodge,
			Target: pos,
		}
	}
	for position, posibility := range radar.ForestThreat {
		if posibility > 0.9 {
			if fire, ok := fireForest[f.Position { X: position.X, Y: position.Y }]; ok {
				objective[fire.tankId] = f.Objective {
					Action: fire.action,
				}
			}
		}
	}
}

func (self *Less) End(state *f.GameState) {
}