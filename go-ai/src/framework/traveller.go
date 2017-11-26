// 行走寻路行动子系统
package framework

import (
	"lib/go-astar"
	"sync"
	"math/rand"
	"fmt"
)

type PathCache struct {
	path []Position
	target Position
	expect *Position
	round int
}

type Traveller struct {
	astar astar.AStar
	cache map[string]*PathCache
	collide map[string]int
	round int
}

func NewTraveller() *Traveller {
	inst := &Traveller {
		astar: nil,
		cache: make(map[string]*PathCache),
		collide: make(map[string]int),
		round: 0,
	}
	return inst
}

func (self *Traveller) CollidedTankInForest(state *GameState) []Position {
	var candidate []Position
	myTankPos := make(map[Position]bool)
	for _, tank := range state.MyTank {
		p := Position {
			X: tank.Pos.X,
			Y: tank.Pos.Y,
		}
		myTankPos[p] = true
		if cache, ok := self.cache[tank.Id]; ok {
			from := &tank.Pos
			if cache.expect != nil && (cache.expect.Y != from.Y || cache.expect.X != from.X) {
				pos := Position {
					X: from.X + sign(cache.expect.X - from.X),
					Y: from.Y + sign(cache.expect.Y - from.Y),
				}
				if state.Terain.Get(pos.X, pos.Y) == 2 {
					candidate = append(candidate, pos)
				}
			}
		}
	}
	var ret []Position
	for _, pos := range candidate {
		if !myTankPos[pos] {
			ret = append(ret, pos)
		}
	}
	return ret
}

func (self *Traveller) Search(travel map[string]*Position, state *GameState, threat map[Position]float64, movements map[string]int) {
	maxPathCalc := 9
	self.round++
	if self.astar == nil {
		self.astar = astar.NewAStar(state.Terain.Height, state.Terain.Width)
		for y := 0; y < state.Terain.Height; y++ {
			for x := 0; x < state.Terain.Width; x++ {
				switch state.Terain.Get(x, y) {
				case 1:
					self.astar.FillTile(astar.Point{ Col: x, Row: y }, -1)
				case 2:
					self.astar.FillTile(astar.Point{ Col: x, Row: y }, 1)
				}
			}
		}
	}
	waitchan := make(chan bool)
	var lock sync.Mutex
	occupy := make(map[Position]bool)
	a := self.astar.Clone()
	lw := 5
	for _, tank := range state.MyTank {
		a.FillTile(astar.Point{ Col: tank.Pos.X, Row: tank.Pos.Y }, lw)
	}
	for _, tank := range state.EnemyTank {
		a.FillTile(astar.Point{ Col: tank.Pos.X, Row: tank.Pos.Y }, lw)
	}
	lock.Lock()
	var myTanks []*Tank
	for _, tank := range state.MyTank {
		if _, exists := travel[tank.Id]; exists {
			t := tank
			myTanks = append(myTanks, &t)
		} else {
			occupy[Position { X: tank.Pos.X, Y: tank.Pos.Y }] = true
			if cache, hasCache := self.cache[tank.Id]; hasCache {
				cache.expect = nil
				cache.path = nil
			}
		}
	}
	if len(myTanks) > maxPathCalc {
		full := myTanks
		myTanks = make([]*Tank, len(full))
		for i, j := range rand.Perm(len(myTanks)) {
			myTanks[i] = full[j]
		}
	}
	firstColide := true
	for _, mtank := range myTanks {
		tank := mtank
		id := tank.Id
		from := tank.Pos
		to := *travel[tank.Id]
		go (func () {
			nextPoint := to
			lock.Lock()
			cache, hasCache := self.cache[id]
			if !hasCache {
				cache = &PathCache {}
				self.cache[id] = cache
			}
			lock.Unlock()
			aThreat := make(map[astar.Point]float64)
			for p, v := range threat {
				if v > 0 {
					aThreat[astar.Point { Col: p.X, Row: p.Y }] = v	
				}
			}
			isDodge := aThreat[astar.Point { Col: from.X, Row: from.Y }] > 0
			// isDodge = true
			if !isDodge {
				directions := []int { DirectionUp, DirectionLeft, DirectionDown, DirectionRight }
				for _, etank := range state.EnemyTank {
					for _, dir := range directions {
						dp := etank.Pos.step(dir)
						if rt, ok := aThreat[astar.Point { Col: dp.X, Row: dp.Y }]; ok && rt < 0.8 {
							aThreat[astar.Point { Col: dp.X, Row: dp.Y }] = 0.8
						}
					}
					var possibles []Position
					possibles = append(possibles, etank.Pos)
					// if tank.Bullet != "" {
						nPos := etank.Pos
						for ti := 0; ti < state.Params.TankSpeed; ti++ {
							nPos = nPos.step(etank.Pos.Direction)
							if state.Terain.Get(nPos.X, nPos.Y) == 1 {
								break
							}
							possibles = append(possibles, nPos)
						}
					// }
					for _, oPos := range possibles {
						for _, dir := range directions {
							pos := oPos
							if abs(pos.X - tank.Pos.X) > state.Params.TankSpeed * 2 && abs(pos.Y - tank.Pos.Y) > state.Params.TankSpeed * 2 {
								continue
							}
							aThreat[astar.Point { Col: pos.X, Row: pos.Y }] = 1
							badDir := false
							if dir == DirectionUp || dir == DirectionDown {
								if tank.Pos.Direction == DirectionUp || tank.Pos.Direction == DirectionDown {
									badDir = true
								}
							} else {
								if tank.Pos.Direction == DirectionLeft || tank.Pos.Direction == DirectionRight {
									badDir = true
								}
							}
							dangerDist := state.Params.BulletSpeed + 2
							for i, N := 1, dangerDist + state.Params.BulletSpeed + 1; i <= N; i++ {
								pos = pos.step(dir)
								if state.Terain.Get(pos.X, pos.Y) == 1 {
									break
								}
								isThreat := false
								if i <= dangerDist {
									isThreat = true
								} else if badDir {
									isThreat = true
								} else {
									fpos := pos.step(tank.Pos.Direction)
									if state.Terain.Get(fpos.X, fpos.Y) == 1 {
										isThreat = true
									}
								}
								if state.Terain.Get(pos.X, pos.Y) == 2 {
									isThreat = false
								}
								if isThreat {
									if rt, ok := aThreat[astar.Point { Col: pos.X, Row: pos.Y }]; ok && rt < 0.6 {
										aThreat[astar.Point { Col: pos.X, Row: pos.Y }] = -1
									}
								}
							}
						}
					}
					fmt.Println("ATHREAT", aThreat)
				}
			} else {
				fmt.Println("IS Dodge")
			}
			if from.X != to.X || from.Y != to.Y {
				cache.path = nil
				if to.SDist(from) <= state.Params.TankSpeed {
					cache.path = []Position { to }
				}
				cache.target = to
				if cache.expect != nil {
					lock.Lock()
					collide := self.collide[tank.Id]
					if cache.expect.Y != from.Y || cache.expect.X != from.X {
						cache.path = nil
						self.collide[tank.Id] = collide + 10
					} else if collide > 0 {
						self.collide[tank.Id] = collide - 1
					}
					lock.Unlock()
				}
				for len(cache.path) > 0 {
					p := cache.path[0]
					if p.X == from.X && p.Y == from.Y {
						cache.path = cache.path[1:]
					} else {
						break
					}
				}
				if len(cache.path) == 0 {
					lock.Lock()
					allowCalc := false
					if maxPathCalc > 0 {
						maxPathCalc--
						allowCalc = true
					}
					lock.Unlock()
					if allowCalc {
						cache.path = self.path(a, from, to, state.Params.TankSpeed, state.Terain, aThreat, aThreat[astar.Point { Col: tank.Pos.X, Row: tank.Pos.Y }] > 0.3)
						for len(cache.path) > 0 {
							p := cache.path[0]
							if p.X == from.X && p.Y == from.Y {
								cache.path = cache.path[1:]
								cache.round = self.round
							} else {
								break
							}
						}
					}
				}
				if len(cache.path) == 0 {
					nextPoint = to
				} else {
					nextPoint = cache.path[0]
				}
			}
			action := toAction(from, nextPoint)
			cache.expect = nil
			lock.Lock()
			if action == ActionMove {
				p := Position { Y: from.Y, X: from.X }
				threatPrevent := false
				thr := 0.
				mp := p
				for i := 0; i < state.Params.TankSpeed; i++ {
					lp := mp.step(tank.Pos.Direction)
					if state.Terain.Get(mp.X, mp.Y) == 1 {
						break;
					}
					mp = lp
					t := aThreat[astar.Point { Col: mp.X, Row: mp.Y }]
					if t > 0 {
						thr += t
					}
				}
				lastThreat := aThreat[astar.Point { Col: mp.X, Row: mp.Y }]
				if lastThreat < 0 {
					thr -= lastThreat
				}
				curThreat := aThreat[astar.Point { Col: tank.Pos.X, Row: tank.Pos.Y }]
				if curThreat > 0.4 {
					threatPrevent = false
				} else {
					threatPrevent = thr > 0.5
				}
				if threatPrevent {
					action = ActionStay
					fmt.Println("Travel threat stay!!")
				} else if _, exists := occupy[p]; exists {
					action = ActionStay
					if firstColide {
						cache.path = nil
						firstColide = false
					}
				} else {
					p = p.step(tank.Pos.Direction)
					cache.expect = &nextPoint
				}
				occupy[p] = true
			} else {
				p := Position { Y: from.Y, X: from.X }
				occupy[p] = true
			}
			movements[id] = action
			lock.Unlock()
			waitchan <- true
		})()
	}
	lock.Unlock()
	for _, _ = range myTanks {
		_ = <- waitchan
	}
}

func (self *Traveller) path(a astar.AStar, source Position, target Position, movelen int, terain *Terain, threat map[astar.Point]float64, brave bool) []Position {
	p2p := astar.NewPointToPoint()

	sourcePoint := []astar.Point{ astar.Point{ Row: source.Y, Col: source.X } }
	targetPoint := []astar.Point{ astar.Point{ Row: target.Y, Col: target.X } }

	p := a.FindPath(p2p, targetPoint, sourcePoint, movelen, source.Direction, threat, brave)
	
	var ret []Position
	for p != nil {
		ret = append(ret, Position {
			X: p.Col,
			Y: p.Row,
		})
		p = p.Parent
	}
	c := len(ret)
	for i, n := 0, c / 2; i < n; i++ {
		j := c - i - 1
		ret[i], ret[j] = ret[j], ret[i]
	}
	return ret
}

func abs (val int) int {
	if val < 0 {
		return -val
	} else {
		return val
	}
}

func sign (val int) int {
	if val > 0 {
		return 1
	} else if val < 0 {
		return -1
	} else {
		return 0
	}
}

func toAction (source Position, target Position) int {
	targetDirection := DirectionNone
	if source.X < target.X {
		targetDirection = DirectionRight
	} else if source.X > target.X {
		targetDirection = DirectionLeft
	} else if source.Y < target.Y {
		targetDirection = DirectionDown
	} else if source.Y > target.Y {
		targetDirection = DirectionUp
	} else {
		targetDirection = target.Direction
		if targetDirection == DirectionNone || source.Direction == target.Direction {
			return ActionStay	
		}
	}
	if targetDirection == source.Direction {
		return ActionMove
	}
	return targetDirection - DirectionUp + ActionTurnUp
}
