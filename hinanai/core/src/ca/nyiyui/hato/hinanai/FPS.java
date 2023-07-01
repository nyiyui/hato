package ca.nyiyui.hato.hinanai;

import com.badlogic.gdx.Gdx;
import com.badlogic.gdx.graphics.Color;
import com.badlogic.gdx.graphics.g2d.Batch;
import com.badlogic.gdx.graphics.g2d.BitmapFont;
import com.badlogic.gdx.scenes.scene2d.Actor;
import com.badlogic.gdx.scenes.scene2d.ui.Label;
import org.w3c.dom.html.HTMLBaseElement;

public class FPS extends Actor {
    private final HinanaiGame game;
    private final Label fps;

    FPS(HinanaiGame game) {
        this.game=game;
        Label.LabelStyle ls = new Label.LabelStyle();
        ls.font = game.debugFont;
        ls.fontColor = new Color(0xffffffff);
        fps = new Label("- fps",ls);
    }

    @Override
    public void draw(Batch batch, float parentAlpha) {
        fps.setText(String.format("%d fps", Gdx.graphics.getFramesPerSecond()));
        fps.draw(batch, parentAlpha);
    }
}
