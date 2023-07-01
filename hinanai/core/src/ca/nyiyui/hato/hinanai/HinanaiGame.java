package ca.nyiyui.hato.hinanai;

import com.badlogic.gdx.ApplicationAdapter;
import com.badlogic.gdx.Game;
import com.badlogic.gdx.Gdx;
import com.badlogic.gdx.graphics.Color;
import com.badlogic.gdx.graphics.g2d.BitmapFont;
import com.badlogic.gdx.graphics.g2d.freetype.FreeTypeFontGenerator;
import com.badlogic.gdx.graphics.g2d.freetype.FreetypeFontLoader;
import com.badlogic.gdx.utils.ScreenUtils;

public class HinanaiGame extends Game {
    FreeTypeFontGenerator font;
    BitmapFont debugFont;

    @Override
    public void create() {
        font = new FreeTypeFontGenerator(Gdx.files.internal("fonts/RobotoMono/RobotoMono-VariableFont_wght.ttf"));
        FreeTypeFontGenerator.FreeTypeFontParameter param = new FreeTypeFontGenerator.FreeTypeFontParameter();
        param.size = 16;
        param.color = new Color(0x000000ff);
        param.borderColor = new Color(0xffffffff);
        param.borderWidth = 2;
        debugFont = font.generateFont(param);
        setScreen(new LineDiagram(this));
    }

    @Override
    public void dispose() {
        font.dispose();
        debugFont.dispose();
    }
}
