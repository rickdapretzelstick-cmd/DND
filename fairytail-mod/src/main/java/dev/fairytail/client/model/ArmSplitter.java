package dev.fairytail.client.model;

import dev.fairytail.mixin.ModelPartAccessor;
import net.minecraft.client.model.geom.ModelPart;
import net.minecraft.core.Direction;

import java.util.EnumSet;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Gives a vanilla player arm a real elbow while keeping the player's own skin.
 *
 * <p>Vanilla draws each arm as ONE 4x12x4 cuboid. We swap it for an upper half (6 px) and add a child
 * part for the forearm (6 px) whose UVs are offset so the lower half samples exactly the same skin
 * pixels the vanilla arm used. Standing still the two halves are pixel-identical to the original arm;
 * only when the forearm is rotated does the elbow bend. (Model-space note: +Y is DOWN, so the Cube
 * "DOWN" polygon is the shoulder cap and "UP" is the hand end.)
 */
public final class ArmSplitter {
    private static final Set<Direction> SIDES = EnumSet.of(Direction.NORTH, Direction.SOUTH, Direction.EAST, Direction.WEST);

    /**
     * @param u     skin texture U offset of the vanilla arm (or sleeve) cuboid
     * @param v     skin texture V offset
     * @param x0    cuboid minimum X relative to the arm pivot (right: -3 classic / -2 slim, left: -1)
     * @param width cuboid width (4 classic, 3 slim)
     * @param grow  deformation (0 for the arm, 0.25 for the sleeve)
     * @return the new forearm part (already attached as a child of {@code arm})
     */
    public static ModelPart split(ModelPart arm, int u, int v, float x0, float width, float grow) {
        // upper half: shoulder cap + 4 sides, no end cap
        ModelPart.Cube upper = new ModelPart.Cube(u, v, x0, -2f, -2f, width, 6f - grow, 4f,
                grow, grow, grow, false, 64f, 64f, upperFaces());
        // forearm: 4 sides only; skin rows (v + 6 + 4 ..) correspond to the lower half of the arm
        ModelPart.Cube lower = new ModelPart.Cube(u, v + 6, x0, 0f, -2f, width, 6f + grow, 4f,
                grow, 0f, grow, false, 64f, 64f, SIDES);
        // hand cap: zero-height cuboid that reuses the vanilla end-cap UV region
        ModelPart.Cube cap = new ModelPart.Cube(u, v, x0, 6f + grow, -2f, width, 0f, 4f,
                grow, 0f, grow, false, 64f, 64f, EnumSet.of(Direction.UP));

        ModelPart forearm = new ModelPart(List.of(lower, cap), new HashMap<>());
        forearm.setPos(0f, 4f, 0f);

        ModelPartAccessor acc = (ModelPartAccessor) (Object) arm;
        acc.fairytail$setCubes(List.of(upper));
        Map<String, ModelPart> kids = new HashMap<>(acc.fairytail$getChildren());
        kids.put("fairytail_forearm", forearm);
        acc.fairytail$setChildren(kids);
        return forearm;
    }

    private static Set<Direction> upperFaces() {
        Set<Direction> s = EnumSet.copyOf(SIDES);
        s.add(Direction.DOWN);
        return s;
    }

    private ArmSplitter() {}
}
