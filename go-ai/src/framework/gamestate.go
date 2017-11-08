package framework

import (
	"encoding/json"
)

type GameState struct {
	Raw []byte
	Ended bool
	Params Params

	Events []Event
	Terain Terain
	FlagWait int
	FlagPos Position
	MyTank, EnemyTank []Tank
	MyBullet, EnemyBullet []Bullet
	MyFlag, EnemyFlag int
}

type Params struct {
	TankSpeed int
	BulletSpeed int
	TankPoint int
	FlagPoint int
	FlagTime int
	// Timeout int
}

type Terain struct {
	Width int
	Height int
	Data [][]int
}

const (
	TerainEmpty = 0
	TerainObstacle = 1
	TerainForest = 2
)

func (self Terain) Get(x int, y int) int {
	if (x < 0 || x >= self.Width || y < 0 || y >= self.Height) {
		return 1
	}
	return self.Data[y][x]
}

type Tank struct {
	Id string
	Hp int
	Pos Position
	Bullet string
}

type Bullet struct {
	Id string
	From string
	Pos Position
}

type Event struct {
	Typ string
	Target string
	From string
}

// 位置（可携带方向）
type Position struct {
	X, Y, Direction int
}

func DirectionFromStr (str string) int {
	switch (str) {
	case "up":
		return DirectionUp;
	case "left":
		return DirectionLeft;
	case "down":
		return DirectionDown;
	case "right":
		return DirectionRight;
	default:
		return DirectionNone;
	}
}

func ActionFromStr (str string) int {
	switch (str) {
	case "stay":
		return ActionStay;
	case "move":
		return ActionMove;
	case "left":
		return ActionLeft;
	case "right":
		return ActionRight;
	case "back":
		return ActionBack;
	case "fire-up":
		return ActionFireUp;
	case "fire-left":
		return ActionFireLeft;
	case "fire-down":
		return ActionFireDown;
	case "fire-right":
		return ActionFireRight;
	default:
		return ActionNone;
	}
}

func ParseGameState (bytes []byte) (*GameState, error) {
	var dat map[string]interface{}
	if err := json.Unmarshal(bytes, &dat); err != nil {
		return nil, err
	}
	ret := &GameState {
		Raw: bytes,
		Terain: Terain {
			Width: 0,
			Height: 0,
			Data: nil,
		},
		MyTank: nil,
		EnemyTank: nil,
		MyBullet: nil,
		EnemyBullet: nil,
		MyFlag: 0,
		EnemyFlag: 0,
		Params: Params {
			TankSpeed: 0,
			BulletSpeed: 0,
			TankPoint: 0,
			FlagPoint: 0,
			FlagTime: 0,
			// Timeout: 1000,
		},
		Events: nil,
		Ended: dat["ended"].(bool),
	}
	// parse terain
	for _, iline := range dat["terain"].([]interface{}) {
		line := iline.([]interface{})
		ret.Terain.Width = len(line)
		oline := make([]int, ret.Terain.Width)
		for i, v := range line {
			oline[i] = int(v.(float64))
		}
		ret.Terain.Data = append(ret.Terain.Data, oline)
	}
	ret.Terain.Height = len(ret.Terain.Data)
	// parse my/enemy game status
	parseTank(dat["myTank"].([]interface{}), &ret.MyTank)
	parseTank(dat["enemyTank"].([]interface{}), &ret.EnemyTank)
	parseBullet(dat["myBullet"].([]interface{}), &ret.MyBullet)
	parseBullet(dat["enemyBullet"].([]interface{}), &ret.EnemyBullet)
	ret.MyFlag = dat["myFlag"].(int)
	ret.EnemyFlag = dat["enemyFlag"].(int)
	// parse params
	params := dat["params"].(map[string]interface{});
	ret.Params.TankSpeed = params["tankSpeed"].(int)
	ret.Params.BulletSpeed = params["bulletSpeed"].(int)
	ret.Params.TankPoint = params["tankPoint"].(int)
	ret.Params.FlagPoint = params["flagPoint"].(int)
	ret.Params.FlagTime = params["flagTime"].(int)
	// parse events
	for _, ievent := range dat["events"].([]interface{}) {
		event := ievent.(map[string]interface{})
		from, _ := event["from"].(string)
		ret.Events = append(ret.Events, Event {
			Typ: event["type"].(string),
			Target: event["target"].(string),
			From: from,
		})
	}
	return ret, nil
}

func parseTank(dat []interface{}, tanks *[]Tank) {
	for _, itank := range dat {
		tank := itank.(map[string]interface{})
		*tanks = append(*tanks, Tank {
			Id: tank["id"].(string),
			Pos: Position {
				X: int(tank["x"].(float64)),
				Y: int(tank["y"].(float64)),
				Direction: DirectionFromStr(tank["direction"].(string)),
			},
		})
	}
}

func parseBullet(dat []interface{}, bullets *[]Bullet) {
	for _, ibullet := range dat {
		bullet := ibullet.(map[string]interface{})
		*bullets = append(*bullets, Bullet {
			Id: bullet["id"].(string),
			From: bullet["from"].(string),
			Pos: Position {
				X: int(bullet["x"].(float64)),
				Y: int(bullet["y"].(float64)),
				Direction: DirectionFromStr(bullet["direction"].(string)),
			},
		})
	}
}
