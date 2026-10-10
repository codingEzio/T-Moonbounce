package main

import "math"

const regionLength = 1800.0

type Biome struct {
	Name, Hans, Hant, Mood, MoodHans, MoodHant string
	Sky                                        int
	Ground, Alternate, Road, Edge, Accent      uint32
	Prop                                       int
}

// Each world changes its silhouettes, roadside objects, road material and atmosphere.
var biomes = []Biome{
	{"Peach Coast", "蜜桃海岸", "蜜桃海岸", "Salt air & nowhere to be", "海风吹过，不赶时间", "海風吹過，不趕時間", 0, 0x4ea8a1, 0x439894, 0x4b4265, 0xffdca8, 0xffb78b, 0},
	{"Alpine Sunday", "阿尔卑斯周日", "阿爾卑斯週日", "Take the long way home", "绕远路，也很好", "繞遠路，也很好", 1, 0x709b94, 0x618c88, 0x58677b, 0xe6e8ce, 0xaee3dc, 1},
	{"Neon Noodles", "霓虹面馆", "霓虹麵館", "The city saved you a seat", "城市为你留了个位子", "城市為你留了個位子", 2, 0x363457, 0x303050, 0x272843, 0xd27fa6, 0x67ddd5, 2},
	{"Moonmilk Way", "月乳银河", "月乳銀河", "A very small car. A very big sky.", "小小的车，大大的天空", "小小的車，大大的天空", 3, 0x846d9e, 0x77618e, 0x55496e, 0xe6b2cd, 0xe2bbf4, 3},
	{"Sakura Shortcut", "樱花小径", "櫻花小徑", "Petals have right of way", "让花瓣先走", "讓花瓣先走", 0, 0xaab189, 0x9ba17d, 0x726781, 0xf3c4cf, 0xf3b5ce, 4},
	{"Canyon Radio", "峡谷电台", "峽谷電台", "Windows down. Worries out.", "打开车窗，放下心事", "打開車窗，放下心事", 0, 0xc48663, 0xb67958, 0x85606a, 0xf4c99a, 0xf0ae76, 5},
	{"Mosswood", "苔藓森林", "苔蘚森林", "Follow the soft green light", "跟着温柔的绿光", "跟著溫柔的綠光", 1, 0x497a68, 0x416b5d, 0x586260, 0xa8bc8c, 0xbbe5a4, 6},
	{"Lavender FM", "薰衣草电台", "薰衣草電台", "Nothing urgent on this frequency", "这个频道，没有急事", "這個頻道，沒有急事", 0, 0x9c8ba9, 0x8c7b9c, 0x665a77, 0xe4c6dd, 0xcac0ef, 7},
	{"Glacier Postcard", "冰川明信片", "冰川明信片", "Cool air. Warm little engine.", "风很凉，心很暖", "風很涼，心很暖", 1, 0xb2d9dd, 0x9ccbd3, 0x6e90a5, 0xe6f1e5, 0xb7edf4, 8},
	{"Cloud Garden", "云上花园", "雲上花園", "Yes, this road floats", "是的，这条路会漂浮", "是的，這條路會漂浮", 3, 0x9ca4c6, 0x8f9abe, 0x8984a5, 0xf0dec7, 0xf2d9bc, 9},
	{"Bamboo Afternoon", "竹林午后", "竹林午後", "Let the afternoon stretch", "让午后慢慢延长", "讓午後慢慢延長", 1, 0x819673, 0x738765, 0x777568, 0xdac99c, 0xc3d395, 10},
	{"Apricot Dunes", "杏色沙丘", "杏色沙丘", "Every dune is a fresh page", "每座沙丘，都是新的一页", "每座沙丘，都是新的一頁", 0, 0xd8aa7b, 0xc99b70, 0xa07f78, 0xf4d6a5, 0xf8c996, 11},
	{"Gumball Orbit", "糖果轨道", "糖果軌道", "Gravity is feeling generous", "今天的重力很温柔", "今天的重力很溫柔", 3, 0xc890b7, 0xb380ab, 0x8b689e, 0xf3c0de, 0xa6e7d0, 12},
	{"Golden Hour Farm", "金色农场", "金色農場", "The sun can wait for you", "太阳愿意等你", "太陽願意等你", 0, 0xb8ad63, 0xa79d58, 0x807063, 0xe9d2a3, 0xf7d579, 13},
	{"Rainy Day Club", "雨天俱乐部", "雨天俱樂部", "A little rain, a lot of glow", "一点雨，一路光", "一點雨，一路光", 2, 0x576b83, 0x4d6078, 0x394961, 0xa6bacc, 0x99dce0, 14},
	{"Aurora Sleepover", "极光之夜", "極光之夜", "Stay for one more sky", "再看一眼天空", "再看一眼天空", 3, 0x667b96, 0x5b6e8a, 0x455474, 0xbad0dd, 0x97e0cb, 15},
}

type DrivingFeel struct {
	Name, Hans, Hant                string
	Speed, Response, Steering, Grip float64
}

var feels = []DrivingFeel{
	{"Sunday", "周日漫游", "週日漫遊", 36, 1.3, 2.3, 5.5},
	{"Rally", "轻快拉力", "輕快拉力", 50, 2.0, 2.9, 3.2},
	{"Cloud", "云朵漂移", "雲朵漂移", 43, .9, 1.65, 1.8},
	{"Comet", "彗星快车", "彗星快車", 68, 1.6, 3.3, 4.0},
}

type Journey struct {
	Seed                  uint64
	Region                uint64
	Offset, Kilometers    float64
	X, Lateral, Speed     float64
	Feel                  int
	Visited               uint32
	Boost                 float64
	Steering, DriftMeters float64
	Passes                uint64
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func newJourney(seed uint64) Journey {
	j := Journey{Seed: seed, Feel: 1}
	j.Visited = 1 << biomeFor(seed, 0)
	return j
}
func mix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}
func random01(x uint64) float64 { return float64(mix64(x)>>11) * (1.0 / (1 << 53)) }
func biomeFor(seed, region uint64) int {
	return int((region*7 + seed%uint64(len(biomes))) % uint64(len(biomes)))
}
func (j *Journey) biome() int     { return biomeFor(j.Seed, j.Region) }
func (j *Journey) roadStyle() int { return int(mix64(j.Seed+j.Region) % 6) }

// Curve control points use integer world knots and smooth interpolation. Only the
// local fraction is a float: long journeys don't lose position precision.
func (j *Journey) curve(ahead float64) float64 {
	position := j.Offset + ahead
	off := math.Floor(position / 300)
	fraction := (position - off*300) / 300
	knot := j.Region*6 + uint64(int64(off))
	v1 := random01(j.Seed^knot)*2 - 1
	v2 := random01(j.Seed^(knot+1))*2 - 1
	blend := fraction * fraction * (3 - 2*fraction)
	return v1 + (v2-v1)*blend
}
func (j *Journey) visit(biome int) {
	for n := uint64(1); n <= uint64(len(biomes)); n++ {
		if biomeFor(j.Seed, j.Region+n) == biome {
			j.Region += n
			j.Offset = 0
			j.Visited |= 1 << biome
			return
		}
	}
}

type Controls struct {
	Steer, Throttle      float64
	Brake, Drift, Cruise bool
}

func (j *Journey) step(dt, steer float64, drift, slow bool) {
	j.drive(dt, Controls{Steer: steer, Drift: drift, Brake: slow, Cruise: true})
}
func (j *Journey) drive(dt float64, in Controls) {
	if dt <= 0 {
		return
	}
	dt = math.Min(dt, .05)
	f := feels[j.Feel]
	target := 0.0
	if in.Cruise {
		target = f.Speed
	}
	if in.Throttle > 0 {
		target = f.Speed * 1.22
	}
	response := f.Response
	if !in.Cruise && in.Throttle == 0 {
		response = .18
	}
	if j.Boost > 0 {
		target = f.Speed * 1.55
		j.Boost = math.Max(0, j.Boost-dt)
		response = 1.9
	}
	if math.Abs(j.X) > 1.03 {
		target *= .62
	}
	if in.Brake {
		target = 0
		response = 3.2
	}
	decay := math.Exp(-response * dt)
	travel := target*dt + (j.Speed-target)*(1-decay)/response
	j.Speed = target + (j.Speed-target)*decay
	if j.Speed < .03 && target == 0 {
		j.Speed = 0
	}
	j.Steering += (in.Steer - j.Steering) * (1 - math.Exp(-dt*9))
	speedFactor := math.Min(j.Speed/18, 1)
	grip := f.Grip
	if in.Drift {
		grip *= .45
	}
	goal := j.Steering * f.Steering * speedFactor / grip
	// The very same curvature generates both the projected road and cornering load.
	corner := j.curve(0) * .24 * math.Pow(j.Speed/50, 2)
	old := j.Lateral
	d := math.Exp(-grip * dt)
	j.Lateral = goal + (old-goal)*d
	j.X += goal*dt + (old-goal)*(1-d)/grip - corner*dt
	if in.Drift && math.Abs(j.Lateral) > .2 {
		j.DriftMeters += travel
	}
	// Soft verges guide the car back; no damage, resets, fines or lost progress.
	if math.Abs(j.X) > 1.05 {
		j.X -= math.Copysign((math.Abs(j.X)-1.05)*dt*2.5, j.X)
	}
	j.X = math.Max(-1.45, math.Min(1.45, j.X))
	j.Offset += travel
	j.Kilometers += travel / 1000
	for j.Offset >= regionLength {
		j.Offset -= regionLength
		j.Region++
		j.Visited |= 1 << j.biome()
	}
}

// A rolling one-kilometer sprint gives the endless road a short racing rhythm.
// Race state is session-local; trip distance, discoveries and passes stay saved.
const sprintLength = 1000.0

type Sprint struct {
	Meters, Nitro     float64
	Number, LastPlace int
}

func (s *Sprint) advance(meters, dt float64, drafting, drifting bool) {
	s.Meters += math.Max(0, meters)
	recharge := .014
	if drafting {
		recharge += .11
	}
	if drifting {
		recharge += .045
	}
	s.Nitro = math.Min(1, math.Max(0, s.Nitro+math.Min(.05, math.Max(0, dt))*recharge))
}
