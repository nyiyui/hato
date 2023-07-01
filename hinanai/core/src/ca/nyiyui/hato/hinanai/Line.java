package ca.nyiyui.hato.hinanai;

import com.badlogic.gdx.graphics.Color;
import com.badlogic.gdx.graphics.g2d.Batch;
import com.badlogic.gdx.graphics.glutils.ShapeRenderer;
import com.badlogic.gdx.math.Rectangle;
import com.badlogic.gdx.math.Vector2;
import com.badlogic.gdx.scenes.scene2d.Actor;
import com.badlogic.gdx.scenes.scene2d.ui.Label;


public class Line extends Actor {
    private final HinanaiGame game;
    private final ShapeRenderer shape;
    private final Label label;
    private float length;
    private String name;

    Line(HinanaiGame game, String name, Vector2 pos, float length) {
        this.game = game;
        this.name = name;
        setPosition(pos.x, pos.y);
        ;
        this.length = length;
        shape = new ShapeRenderer();
        Label.LabelStyle ls = new Label.LabelStyle();
        ls.font = game.debugFont;
        label = new Label(name, ls);
    }

    @Override
    public void draw(Batch batch, float parentAlpha) {
        label.draw(batch,parentAlpha);
        batch.end();
        shape.setProjectionMatrix(batch.getProjectionMatrix());
        shape.setColor(new Color(0xffffffff));
        shape.begin(ShapeRenderer.ShapeType.Filled);
        shape.rect(0, 0, length, .1f);
        shape.end();
        batch.begin();
    }
}
