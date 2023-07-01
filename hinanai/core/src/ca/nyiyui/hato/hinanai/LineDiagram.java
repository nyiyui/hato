package ca.nyiyui.hato.hinanai;

import com.badlogic.gdx.Gdx;
import com.badlogic.gdx.Input;
import com.badlogic.gdx.ScreenAdapter;
import com.badlogic.gdx.graphics.GL20;
import com.badlogic.gdx.graphics.OrthographicCamera;
import com.badlogic.gdx.graphics.g2d.SpriteBatch;
import com.badlogic.gdx.math.Vector2;
import com.badlogic.gdx.scenes.scene2d.Stage;
import com.badlogic.gdx.utils.viewport.*;

import javax.swing.text.View;
import java.util.ArrayList;

public class LineDiagram extends ScreenAdapter {
    private final HinanaiGame game;
    private final OrthographicCamera cam;
    private final Stage stage;
    private final Viewport viewport;
    private final SpriteBatch batch;
    private final Viewport uiViewport;
    private final Stage uiStage;
    private ArrayList<Line> lines;
    private FPS fps;

    LineDiagram(HinanaiGame game) {
        this.game = game;
        float x = 8;
        float y = 8;
        cam = new OrthographicCamera();
        cam.setToOrtho(false, x, y);
        viewport = new FillViewport(x, y, cam);
        batch = new SpriteBatch();
        stage = new Stage(viewport, batch);
        uiViewport = new ScreenViewport();
        uiStage = new Stage(uiViewport, batch);
        fps = new FPS(game);
        uiStage.addActor(fps);
        lines = new ArrayList<>();
        lines.add(new Line(game, "A", new Vector2(0, 0), .560f));
        lines.add(new Line(game, "B", new Vector2(.560f, 0), .560f));
        lines.add(new Line(game, "C", new Vector2(.560f * 2, 0), .560f));
        lines.add(new Line(game, "D", new Vector2(.560f * 3, 0), .560f));
        for (Line line : lines)
            stage.addActor(line);
    }

    @Override
    public void render(float delta) {
        Gdx.gl.glClearColor(0, 0, 0, 0);
        Gdx.gl.glClear(GL20.GL_COLOR_BUFFER_BIT);
        handleMovement();
        stage.getBatch().setProjectionMatrix(cam.combined);
        stage.act(delta);
        stage.draw();
    }

    private void handleMovement() {
        if (Gdx.input.isKeyPressed(Input.Keys.A)) {
            cam.zoom += 0.02;
        }
        if (Gdx.input.isKeyPressed(Input.Keys.Q)) {
            cam.zoom -= 0.02;
        }
        if (Gdx.input.isKeyPressed(Input.Keys.LEFT)) {
            cam.translate(-3*cam.zoom, 0, 0);
        }
        if (Gdx.input.isKeyPressed(Input.Keys.RIGHT)) {
            cam.translate(3*cam.zoom, 0, 0);
        }
        if (Gdx.input.isKeyPressed(Input.Keys.DOWN)) {
            cam.translate(0, -3*cam.zoom, 0);
        }
        if (Gdx.input.isKeyPressed(Input.Keys.UP)) {
            cam.translate(0, 3*cam.zoom, 0);
        }
        cam.update();
    }

    @Override
    public void resize(int width, int height) {
        uiStage.getViewport().update(width, height, true);
        cam.viewportWidth = 20f;
        cam.viewportHeight = 20f * height / width;
        cam.update();
        fps.setPosition(0, height);
    }

    @Override
    public void dispose() {
        stage.dispose();
    }
}
