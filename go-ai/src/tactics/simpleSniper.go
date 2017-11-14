package tactics

import (
	f "framework"
    // "fmt"
)

type SimpleSniper struct {
    superior *Simple
    tanks    map[string]f.Tank
    policy   *SimplePolicy
	obs      *Observation
}

// 初始化
func (s *SimpleSniper) Init(tanks []f.Tank) {
    s.tanks = make(map[string]f.Tank)
    for _, tank := range tanks {
        s.tanks[tank.Id] = tank
    }
}

// 制定执行计划
func (s *SimpleSniper) Plan(state *f.GameState, objs map[string]f.Objective){
	// 本次可支配的坦克
	var tanks []f.Tank
	for _, tank := range state.MyTank {
        if s.tanks[tank.Id] != (f.Tank{}) {
            if objs[tank.Id] == (f.Objective{}) {
    			tanks = append(tanks, tank)
    		}
        }
	}
	if len(tanks) > 0 {
		s.HideAndFire(tanks, objs)
	}
}

func (s *SimpleSniper) HideAndFire(tanks []f.Tank, objs map[string]f.Objective) {
    // 按距离远近分配行动目标
	arrPos := SortByPos(s.obs.Flag.Pos, s.obs.Kps[0:len(tanks)])    // 优先攻击旗点附近坦克
	ftanks := s.policy.Dispatch(tanks, arrPos[0:len(tanks)], objs)

    // 已抵达目的地坦克，根据友伤选择是否开火
    var radarFire f.RadarFireAll
    for _, ftank := range ftanks {
        radarFire = s.obs.Radar.Fire[ftank.Id]

        if s.obs.Side == "red" {
            if radarFire == (f.RadarFireAll{}) || radarFire.Down.Sin <= 0.3 {
                objs[ftank.Id] = f.Objective{ Action: f.ActionFireDown}
            } else {
                objs[ftank.Id] = f.Objective{ Action: f.ActionStay}
            }
        } else {
            if radarFire == (f.RadarFireAll{}) || radarFire.Up.Sin <= 0.3 {
                objs[ftank.Id] = f.Objective{ Action: f.ActionFireUp}
            } else {
                objs[ftank.Id] = f.Objective{ Action: f.ActionStay}
            }
        }
    }
}