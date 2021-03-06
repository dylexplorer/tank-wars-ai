package tactics

import (
	"fmt"
	f "framework"
	// "fmt"
    "math/rand"
)

type CattyRole struct {
    gotoforest   bool
    gotoflag     bool
    forest       f.Forest
    obs          *Observation
    Tank         f.Tank
    Target       f.Position
    Dodge        f.RadarDodge     // 躲避建议
    Fire         f.RadarFireAll   // 开火建议
    ExtDangerSrc []f.ExtDangerSrc   // 躲不掉和火线上的威胁源
}

// 草内巡逻
func (r *CattyRole) patrol() {
    // 夺旗
    if r.forest.HasFlag && r.obs.State.FlagWait < 5 {
        fmt.Println("Get Flag", r.forest.HasFlag, r.obs.Flag.Next)
        r.occupyFlag()
    } else {
        // 必死
        if r.Dodge.Threat == -1 {
            r.obs.Objs[r.Tank.Id] = f.Objective { Action: r.fireBeforeDying() }

        // 可开火
        } else if r.doFireInForest() != -1 && r.Dodge.Threat < 1 {
            r.obs.Objs[r.Tank.Id] = f.Objective { Action: r.doFireInForest() }

        // 可朝旗开火（加入随机量，避免太频繁）
        // } else if r.canFireToFlag() && rand.Int() % 3 == 0 {
        //     r.fireToFlag()
        } else {
            if r.obs.mapanalysis.GetForestByPos(r.Tank.Pos).Id == r.forest.Id {
                fmt.Println("In Forest")
                randomFire := r.selectRandomFire(12)
                if randomFire != nil {
                    r.obs.Objs[r.Tank.Id] = f.Objective { Action: randomFire.Action }
                } else {
                    pos := forestPartol(r.Tank.Pos, r.obs.Terain, r.obs.State.Params.TankSpeed)
                    fmt.Println("Patrol", pos)
                    r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionTravel, Target: pos }
                }
            } else {
                fmt.Println("On the way")
                r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionTravelWithDodge, Target: nfEntrance(r.Tank.Pos, r.forest) }
            }
        }
    }
}

// 草丛最近入口
func nfEntrance(pos f.Position, forest f.Forest) f.Position {
    var target f.Position
    dist := -1
    for p, _ := range forest.ForestMap {
        if nd := pos.SDist(p); dist < 0 || nd < dist {
            dist   = nd
            target = p
        }
    }
    return target
}


func (r *CattyRole) selectRandomFire(rarity int) *f.RadarFire {
    var randomFire *f.RadarFire
    if rand.Int() % rarity == 0 {
        var canFire []*f.RadarFire
        for _, fire := range []*f.RadarFire { r.Fire.Up, r.Fire.Left, r.Fire.Down, r.Fire.Right } {
            if fire != nil && fire.Sin < 0.9 && fire.Cost > 0 && fire.Cost < 5 {
                dir := fire.Action - f.ActionFireUp + f.DirectionUp
                bPos := r.Tank.Pos.Step(dir)
                if r.obs.State.Terain.Get(bPos.X, bPos.Y) != 2 {
                    continue
                }
                if fire.Cost > 1 {
                    canFire = append(canFire, fire)
                }
                if fire.Cost > 2 {
                    canFire = append(canFire, fire)
                    if fire.Action == f.ActionFireRight {
                        canFire = append(canFire, fire)
                        canFire = append(canFire, fire)
                        canFire = append(canFire, fire)
                    }
                    if fire.Action == f.ActionFireDown {
                        canFire = append(canFire, fire)
                        canFire = append(canFire, fire)
                        canFire = append(canFire, fire)
                    }
                }
                canFire = append(canFire, fire)
            }
        }
        if len(canFire) > 0 {
            randomFire = canFire[rand.Int() % len(canFire)]
        }
    }
    return randomFire
}

// 抢旗
func (r *CattyRole) occupyFlag() {
    // 光荣弹
    if r.Dodge.Threat == -1 {
        r.obs.Objs[r.Tank.Id] = f.Objective { Action: r.fireBeforeDying() }

    // faith == 1
    } else if action := r.fireByFaith(0.9, 0.5); action > 0 {
        r.obs.Objs[r.Tank.Id] = f.Objective { Action: action }

    // 寻路
    } else {
        travel := f.ActionTravel
        if r.Dodge.Threat == 1 {
            travel = f.ActionTravelWithDodge
        }
        r.obs.Objs[r.Tank.Id] = f.Objective {
            Action: travel,
            Target: r.obs.Flag.Pos,
        }
    }
}

// 孤单地守旗
func (r *CattyRole) occupyFlagAlone() {
    // 光荣弹
    if r.Dodge.Threat == -1 {
        r.obs.Objs[r.Tank.Id] = f.Objective { Action: r.fireBeforeDying() }

    // faith == 1
    } else if action := r.fireByFaith(0.6, 1); action > 0 {
        r.obs.Objs[r.Tank.Id] = f.Objective { Action: action }

    // 寻路
    } else {
        travel := f.ActionTravel
        if r.Dodge.Threat == 1 {
            travel = f.ActionTravelWithDodge
        }
        r.obs.Objs[r.Tank.Id] = f.Objective {
            Action: travel,
            Target: r.obs.Flag.Pos,
        }
    }
}


// 寻找追击点
func (r *CattyRole) hunt() {
    // 如果没有绝杀点
    if len(r.obs.ShotPos) == 0 {
        // 距离最近的坦克
        ttank := r.neareastEmy()
        // 如果它在安全距离外，直接定位到它身上
        if nd := r.Tank.Pos.SDist(ttank.Pos); nd > r.obs.State.Params.BulletSpeed * 2 {    // TODO 中心点附近
            r.Target = ttank.Pos
            return
        }
    }

    // 如果有绝杀点，去最近的绝杀点
    var tpos f.Position
    dist  := -1
    for pos, tankid := range r.obs.ShotPos {
        nd := r.Tank.Pos.SDist(pos)
        // 增加夹角概率
        if r.obs.Objs[tankid] != (f.Objective{}) {
            nd -= r.obs.State.Params.BulletSpeed
        }
        if dist < 0 || nd < dist {
            fmt.Println("tankid:", tankid)
            dist  = nd
            tpos  = pos
        }
    }
    fmt.Println("dist:", dist)
    fmt.Println("tpos:", tpos)

    // tpos 可能为空
    if tpos != (f.Position{}) {
        r.Target = tpos
        delete(r.obs.ShotPos, tpos)
        // // 删除已被选择的点
        // if index == 0 {
        //     r.obs.ShotPos = r.obs.ShotPos[index+1:]
        // } else if index == len(r.obs.ShotPos) {
        //     r.obs.ShotPos = r.obs.ShotPos[:index]
        // } else {
        //     r.obs.ShotPos = append(r.obs.ShotPos[:index], r.obs.ShotPos[index+1:]...)
        // }
    } else {
        r.Target = r.Tank.Pos         // 原地躲避
    }
}

// 追击
func (r *CattyRole) act() {
    // 必死
	if r.Dodge.Threat == -1 {
		r.obs.Objs[r.Tank.Id] = f.Objective { Action: r.fireBeforeDying() }

    // 可开火
	} else if r.doFire() != -1 && r.Dodge.Threat < 1 {
		r.obs.Objs[r.Tank.Id] = f.Objective { Action: r.doFire() }

    // 可朝旗开火（加入随机量，避免太频繁）
    } else if r.canFireToFlag() && rand.Int() % 3 == 0 {
        r.fireToFlag()

    // 其余情况寻路
	} else {
		r.move()
	}
}

func (r *CattyRole) move() {
    r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionTravelWithDodge, Target: r.Target }
}

func (r *CattyRole) neareastEmy() f.Tank {
    dist := -1
    var ttank f.Tank
    for _, tank := range r.obs.EmyTank {
        if nd:= r.Tank.Pos.SDist(tank.Pos); dist < 0 || nd < dist {
            dist  = nd
            ttank = tank
        }
    }
    return ttank
}

// 光荣弹开火逻辑
func (r *CattyRole) fireBeforeDying() int {
    var mrf *f.RadarFire
    for _, rf := range []*f.RadarFire{ r.Fire.Up, r.Fire.Down, r.Fire.Left, r.Fire.Right } {
        if rf == nil { continue }
        if mrf == nil || mrf.Faith - mrf.Sin < rf.Faith - rf.Sin {
            mrf = rf
        }
    }

    // 无可开火方向，朝子弹来源方向开火
    if mrf == nil || mrf.Faith <= 0  {
        // 威胁子弹来源
        direct := -1
        for _, extd := range r.ExtDangerSrc {
            if extd.Type == f.BULLET_THREAT && extd.Urgent == -1 {
                direct = extd.SourceDir
                break
            }
        }

        // 判断能否发出光辉弹
        var action int
        switch direct {
        case f.DirectionUp:
            mrf    = r.Fire.Up
            action = f.ActionFireUp
        case f.DirectionDown:
            mrf    = r.Fire.Down
            action = f.ActionFireDown
        case f.DirectionRight:
            mrf    = r.Fire.Left
            action = f.ActionFireRight
        case f.DirectionLeft:
            mrf    = r.Fire.Left
            action = f.ActionFireLeft
        }
        if mrf == nil || mrf.Sin < 0.5 {
            return action
        } else {
            return -1
        }

    // 有可开火方向
    } else {
        if mrf.Sin < 0.5 {
            return mrf.Action
        } else {
            return -1
        }
    }
}

// 极高的命中信仰才开炮
func (r *CattyRole) fireByFaith(faith float64, sin float64) int {
    var mrf *f.RadarFire
    for _, rf := range []*f.RadarFire{ r.Fire.Up, r.Fire.Down, r.Fire.Left, r.Fire.Right } {
        if rf != nil && rf.Faith >= faith && rf.Sin < sin {
            mrf = rf
        }
    }
    if mrf == nil {
        return -1
    } else {
        return mrf.Action
    }
}

// 是否有合适的开火方向
func (r *CattyRole) doFire() int {
    var mrf *f.RadarFire
    for _, rf := range []*f.RadarFire{ r.Fire.Up, r.Fire.Down, r.Fire.Left, r.Fire.Right } {
        if rf == nil || !rf.IsStraight { continue }
        if mrf == nil || mrf.Faith - mrf.Sin < rf.Faith - rf.Sin {
            mrf = rf
        }
    }
	if mrf == nil || mrf.Faith < 0.6 || mrf.Sin >= 0.5 {
		return -1
	} else {
		return mrf.Action
	}
}

// 草丛内是否有合适的开火方向
func (r *CattyRole) doFireInForest() int {
    var mrf *f.RadarFire
    for _, rf := range []*f.RadarFire{ r.Fire.Up, r.Fire.Down, r.Fire.Left, r.Fire.Right } {
        if rf == nil || !rf.IsStraight { continue }
        if mrf == nil || mrf.Faith < rf.Faith {
            mrf = rf
        }
    }
	if mrf == nil || mrf.Faith < 0.6 || mrf.Sin >= 0.5 {
		return -1
	} else {
		return mrf.Action
	}
}

// 是否可以朝旗子开火
func (r *CattyRole) canFireToFlag() bool {
    if r.Tank.Pos.X == r.obs.Flag.Pos.X || r.Tank.Pos.Y == r.obs.Flag.Pos.Y {
        // 自己在旗子中不开火
        if (r.Tank.Pos.X == r.obs.Flag.Pos.X && r.Tank.Pos.Y == r.obs.Flag.Pos.Y){
            return false
        }

        // 可以向旗子开火
        if r.Dodge.Threat == 0 && r.obs.pathReachable(r.Tank.Pos, r.obs.Flag.Pos) {
            // 判断友伤
            var rf *f.RadarFire
            if r.Tank.Pos.X == r.obs.Flag.Pos.X {
                if r.Tank.Pos.Y > r.obs.Flag.Pos.Y {
                    rf = r.Fire.Up
                } else {
                    rf = r.Fire.Down
                }
            } else {
                if r.Tank.Pos.X > r.obs.Flag.Pos.X {
                    rf = r.Fire.Left
                } else {
                    rf = r.Fire.Right
                }
            }
            if rf != nil && rf.Sin <= 0.2 {
                return true
            }
        }
    }
    return false
}

// 向旗子开火
func (r *CattyRole) fireToFlag() {
    if r.Tank.Pos.X == r.obs.Flag.Pos.X {
        if r.Tank.Pos.Y > r.obs.Flag.Pos.Y {
            r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionFireUp }
        } else {
            r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionFireDown }
        }
    } else {
        if r.Tank.Pos.X > r.obs.Flag.Pos.X {
            r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionFireLeft }
        } else {
            r.obs.Objs[r.Tank.Id] = f.Objective { Action: f.ActionFireRight }
        }
    }
}
