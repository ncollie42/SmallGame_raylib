package main

import (
	"context"
	"flag"
	"fmt"
	myProto "game/proto"
	"io"
	"log"
	"time"

	r "github.com/lachee/raylib-goplus/raylib"
	"google.golang.org/grpc"
)

type player struct {
	playerCellX, playerCellY int32
	pos                      r.Vector2
}

type world struct {
	model     *r.Model
	position  r.Vector3
	imgPixels []r.Color
	texture   r.Texture2D //cubicMap
}

type game struct {
	cam     r.Camera
	main    player
	world   world
	players *myProto.AllPlayers
}

//TODO:
//basic multiplayer
//see each other
// Make update server a go routines? Make more concurent model?
//test to see if we can use nrock? or digital ocean?
// ------ Next:
// (update server) -> get map of players -> loop over players and draw them -> figure out orientation?
// make Server playable? playable?

func init() {
	flag.StringVar(&defaultName, "name", "default", "name of player")
	flag.StringVar(&defaultIP, "ip", "localhost:8080", "ip:port of server")
	flag.Parse()

}

func initGame() game {
	// Initialization
	//--------------------------------------------------------------------------------------
	r.InitWindow(800, 450, "Raylib Go Plus - Hellow World") //Creates the window
	game := game{}
	// Define the camera to look into our 3d world
	game.cam = r.Camera{Position: r.Vector3{.2, .4, .2}, Target: r.Vector3{0, 0, 0}, Up: r.Vector3{0, 1, 0}, FOVY: 45, Type: r.CameraTypePerspective}

	imMap := r.LoadImage("../resources/cubicmap.png")
	game.world.imgPixels = imMap.GetPixels() //for collision
	game.world.texture = r.LoadTextureFromImage(imMap)
	mesh := r.GenMeshCubicmap(imMap, r.Vector3{1, 1, 1}) //unload?
	game.world.model = r.LoadModelFromMesh(mesh)
	// NOTE: By default each cube is mapped to one part of texture atlas
	texture := r.LoadTexture("../resources/cubicmap_atlas.png")
	game.world.model.Materials[0].SetTexture(r.MapAlbedo, texture)
	game.world.position = r.Vector3{-16.0, 0.0, -8.0} // Set model position

	r.UnloadImage(imMap) // Unload image from RAM
	r.SetCameraMode(&game.cam, r.CameraFirstPerson)
	r.SetTargetFPS(60) // Set our game to run at 60 frames-per-second
	return game
}

var defaultName string
var defaultIP string

func initServer() (myProto.UpdateStateClient, *grpc.ClientConn) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(defaultIP, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := myProto.NewUpdateStateClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	{
		tmp := float32(0)
		myP := myProto.Player{Name: &defaultName, Location: &myProto.Player_Cord{X: &tmp, Y: &tmp}}
		r, err := c.Join(ctx, &myP)
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}
		status := r.GetPlayerMap()
		for name, tt := range status {
			fmt.Println(name, *tt)
		}
	}
	return c, conn
}

func main() {
	game := initGame()
	defer r.CloseWindow() // Close window and OpenGL context
	server, con := initServer()
	defer con.Close()
	defer func() {
		tmp := float32(0)
		myP := myProto.Player{Name: &defaultName, Location: &myProto.Player_Cord{X: &tmp, Y: &tmp}}
		ctx2, cancle := context.WithTimeout(context.Background(), time.Second)
		defer cancle() // <-- panic if I don't
		server.Leave(ctx2, &myP)
		fmt.Println(server)
	}()

	// go func() {
	// 	t := time.NewTicker(time.Second * 2)
	// 	for {
	// 		select {
	// 		case <-t.C:
	// 			fmt.Printf("%#v\n", game.cam)
	// 		}
	// 	}
	// }()
	// defer r.UnloadAll()
	//--------------------------------------------------------------------------------------
	// Main stream of data from and to server (player movment) (server returns all locations)
	go func() {
		ctx, cancle := context.WithCancel(context.Background())
		defer cancle()
		stream, err := server.ConstUpdate(ctx)
		if err != nil {
			fmt.Println("Err with stream?")
		}
		for {
			allObjs, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("%v.ListFeatures(_) = _, %v", server, err)
			}
			log.Println("IN CONST:\n\n", allObjs)
		}
	}()
	// Main game loop
	for !r.WindowShouldClose() { // Detect window close button or ESC key
		// Update
		//----------------------------------------------------------------------------------
		update(&game)
		updateServer(&game, server)
		//----------------------------------------------------------------------------------

		// Draw
		//----------------------------------------------------------------------------------
		draw(&game)
		//----------------------------------------------------------------------------------
	}

	// De-Initialization
	//--------------------------------------------------------------------------------------
	// defer r.UnloadTexture(gcubicmap) // Unload cubicmap texture
	// defer r.UnloadTexture(texture)  // Unload map texture
	// defer r.UnloadModel(model)      // Unload map model

	//--------------------------------------------------------------------------------------
}

//update the server and return the map of players -> then loop over them, skip self and draw
func updateServer(g *game, server myProto.UpdateStateClient) {
	ctx, cancle := context.WithTimeout(context.Background(), time.Second)
	defer cancle()
	myP := myProto.Player{Name: &defaultName, Location: &myProto.Player_Cord{X: &g.cam.Position.X, Y: &g.cam.Position.Z}}
	players, err := server.Update(ctx, &myP)
	if err != nil {
		// fmt.Println("Could not update server" + err.Error()) //Don't panic in the future
	} else {
		// fmt.Println("IT'WORKING!!!!")
		g.players = players
	}
}

func update(g *game) { //Needs cam, mapPosition, cubicmap, mapPixels,
	oldCamPos := g.cam.Position // Store old camera position

	r.UpdateCamera(&g.cam) // Update camera

	// Check player collision (we simplify to 2D collision detection)
	playerPos := r.Vector2{g.cam.Position.X, g.cam.Position.Z}
	playerRadius := float32(0.1) // Collision radius (player is modelled as a cilinder for collision)

	//mini map cube / working cell
	{
		g.main.playerCellX = (int32)(playerPos.X - g.world.position.X + 0.5)
		g.main.playerCellY = (int32)(playerPos.Y - g.world.position.Z + 0.5)

		// Out-of-limits security check
		if g.main.playerCellX < 0 {
			g.main.playerCellX = 0

		} else if g.main.playerCellX >= g.world.texture.Width {
			g.main.playerCellX = g.world.texture.Width - 1
		}
		if g.main.playerCellY < 0 {
			g.main.playerCellY = 0
		} else if g.main.playerCellY >= g.world.texture.Height {
			g.main.playerCellY = g.world.texture.Height - 1
		}
	}
	// Check map collisions using image data and player position
	// TODO: Improvement: Just check player surrounding cells for collision
	for y := int32(0); y < g.world.texture.Height; y++ {
		for x := int32(0); x < g.world.texture.Width; x++ {
			if g.world.imgPixels[y*g.world.texture.Width+x].R == 255 && // Collision: white pixel, only check R channel
				(r.CheckCollisionCircleRec(playerPos, playerRadius,
					r.Rectangle{g.world.position.X - 0.5 + float32(x), g.world.position.Z - 0.5 + float32(y), 1.0, 1.0})) {
				// Collision detected, reset cam position
				g.cam.Position = oldCamPos
			}
		}
	}
}

func draw(g *game) { //cam ,model, mapPosition, playerPosition, cubicMap, playerCell
	r.BeginDrawing()

	r.ClearBackground(r.RayWhite)

	r.BeginMode3D(g.cam)

	r.DrawModel(*g.world.model, g.world.position, 1.0, r.White) // Draw maze map
	// draw players in map
	{
		for name, val := range g.players.PlayerMap {
			if name != defaultName {
				pos := r.Vector3{val.GetLocation().GetX(), .4, val.GetLocation().GetY()}
				r.DrawCubeV(pos, r.Vector3{0.2, 0.4, 0.2}, r.Red) // Draw player
			}
		}
		// r.DrawCubeV(playerPosition, r.Vector3{0.2, 0.4, 0.2}, r.Red) // Draw player
		// playerPosition = g.cam.Position
	}
	r.EndMode3D()

	// Mini map
	{
		r.DrawTextureEx(g.world.texture, r.Vector2{float32(r.GetScreenWidth() - int(g.world.texture.Width*4) - 20), 20}, 0.0, 4.0, r.White)
		r.DrawRectangleLines(r.GetScreenWidth()-int(g.world.texture.Width*4)-20, 20, int(g.world.texture.Width*4), int(g.world.texture.Height*4), r.Green)
		// Draw player position radar
		r.DrawRectangle(r.GetScreenWidth()-int(g.world.texture.Width*4)-20+int(g.main.playerCellX)*4, int(20+g.main.playerCellY*4), 4, 4, r.Red)
	}

	r.DrawFPS(10, 10)

	r.EndDrawing()
}

func drawUI(g *game) {

}
