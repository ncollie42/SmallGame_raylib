package main

import (
	r "github.com/lachee/raylib-goplus/raylib"
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
	cam   r.Camera
	main  player
	world world
}

func initGame() game {
	// Initialization
	//--------------------------------------------------------------------------------------
	r.InitWindow(800, 450, "Raylib Go Plus - Hellow World") //Creates the window
	game := game{}
	// Define the camera to look into our 3d world
	game.cam = r.Camera{Position: r.Vector3{.2, .4, .2}, Target: r.Vector3{0, 0, 0}, Up: r.Vector3{0, 1, 0}, FOVY: 45, Type: r.CameraTypePerspective}

	imMap := r.LoadImage("resources/cubicmap.png")
	game.world.imgPixels = imMap.GetPixels() //for collision
	game.world.texture = r.LoadTextureFromImage(imMap)
	mesh := r.GenMeshCubicmap(imMap, r.Vector3{1, 1, 1}) //unload?
	game.world.model = r.LoadModelFromMesh(mesh)
	// NOTE: By default each cube is mapped to one part of texture atlas
	texture := r.LoadTexture("resources/cubicmap_atlas.png")
	game.world.model.Materials[0].SetTexture(r.MapAlbedo, texture)
	game.world.position = r.Vector3{-16.0, 0.0, -8.0} // Set model position

	r.UnloadImage(imMap) // Unload image from RAM
	r.SetCameraMode(&game.cam, r.CameraFirstPerson)
	r.SetTargetFPS(60) // Set our game to run at 60 frames-per-second
	return game
}

func main() {
	game := initGame()
	defer r.CloseWindow() // Close window and OpenGL context
	// defer r.UnloadAll()
	//--------------------------------------------------------------------------------------

	// Main game loop
	for !r.WindowShouldClose() { // Detect window close button or ESC key
		// Update
		//----------------------------------------------------------------------------------
		update(&game)
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

func update(g *game) { //Needs cam, mapPosition, cubicmap, mapPixels,
	oldCamPos := g.cam.Position // Store old camera position

	r.UpdateCamera(&g.cam) // Update camera

	// Check player collision (we simplify to 2D collision detection)
	playerPos := r.Vector2{g.cam.Position.X, g.cam.Position.Z}
	playerRadius := float32(0.1) // Collision radius (player is modelled as a cilinder for collision)

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

	// r.DrawCubeV(playerPosition, r.Vector3{0.2, 0.4, 0.2}, r.Red) // Draw player
	// playerPosition = g.cam.Position
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
