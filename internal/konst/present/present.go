package present

const (
	TilesPerWorker = 128
	TileArea       = 80 * 60
	PixelsPerTile  = 175
	RecipeBytes    = 32
	ByteLanes      = 0x00ff00ff
	HalfLanes      = 0x00800080
	ShareSlack     = 2
	ShareFar       = 4
	ShareGrid      = 256
	ShareLimit     = 1 << 20
)
