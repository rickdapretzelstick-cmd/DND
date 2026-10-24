package dev.fairytail.client;

import dev.fairytail.FairyTailMod;
import dev.fairytail.key.ZodiacKey;
import net.minecraft.client.CameraType;
import net.minecraft.client.Minecraft;
import net.minecraft.client.Screenshot;
import net.minecraft.world.InteractionHand;

import java.util.Arrays;

/**
 * Reproducible in-game test, enabled with JVM properties (see README):
 * <pre>
 *   -Dfairytail.autotest=aquarius          key to cast (or "all" to cast each key in turn)
 *   -Dfairytail.shots=8,16,28,...          timeline ticks at which to save a screenshot
 *   -Dfairytail.camera=first|back|front    starting camera perspective
 *   -Dfairytail.repeat=2                   casts per key (checks replays do not corrupt state)
 *   -Dfairytail.nofx=true                  hide particles/gate so the pose and key can be inspected
 * </pre>
 * It joins a world, gives itself the key and right-clicks through the REAL item path (server validation,
 * packets, client playback), screenshots the requested ticks, checks the camera/pose restored, then exits.
 */
public final class AutoTest {
    private static final String KEYSPEC = System.getProperty("fairytail.autotest");
    private static final int[] SHOTS = Arrays.stream(System.getProperty("fairytail.shots", "8,16,28,40,48,54,58,62,64,68,76,90").split(","))
            .mapToInt(s -> Integer.parseInt(s.trim())).toArray();
    private static final String CAMERA = System.getProperty("fairytail.camera", "first");
    private static final int REPEAT = Integer.getInteger("fairytail.repeat", 1);

    private enum Phase { WAIT_WORLD, GIVE, CAST, DURING, AFTER, DONE }

    private static Phase phase = Phase.WAIT_WORLD;
    private static int wait;
    private static int keyIdx;
    private static int rep;
    private static int nextShot;
    private static float lastT;
    private static float[] pendingShot = null;

    /** -Dfairytail.nofx=true hides particles, gate and halo so pose / key orientation can be inspected alone. */
    public static final boolean NO_FX = Boolean.getBoolean("fairytail.nofx");

    public static boolean enabled() {
        return KEYSPEC != null;
    }

    private static ZodiacKey current() {
        return "all".equalsIgnoreCase(KEYSPEC) ? ZodiacKey.byIndex(keyIdx) : ZodiacKey.valueOf(KEYSPEC.toUpperCase());
    }

    public static void tick() {
        if (!enabled()) return;
        Minecraft mc = Minecraft.getInstance();
        if (mc.player == null || mc.level == null || mc.gameMode == null) return;
        mc.getToasts().clear();
        switch (phase) {
            case WAIT_WORLD -> {
                if (++wait > 80) {
                    mc.options.setCameraType(switch (CAMERA) {
                        case "back" -> CameraType.THIRD_PERSON_BACK;
                        case "front" -> CameraType.THIRD_PERSON_FRONT;
                        default -> CameraType.FIRST_PERSON;
                    });
                    if (Boolean.getBoolean("fairytail.left")) mc.options.mainHand().set(net.minecraft.world.entity.HumanoidArm.LEFT);
                    mc.player.connection.sendCommand("gamemode creative");
                    phase = Phase.GIVE;
                    wait = 0;
                }
            }
            case GIVE -> {
                if (wait == 3) mc.player.connection.sendCommand("clear @s");
                if (++wait == 8) mc.player.connection.sendCommand("give @s fairytail:" + current().itemName());
                if (wait > 25) {
                    mc.player.getInventory().selected = 0;
                    phase = Phase.CAST;
                    wait = 0;
                }
            }
            case CAST -> {
                mc.gameMode.useItem(mc.player, InteractionHand.MAIN_HAND);
                phase = Phase.DURING;
                nextShot = 0;
                lastT = 0;
                wait = 0;
                log("cast started key=" + current() + " rep=" + (rep + 1) + " camera=" + mc.options.getCameraType());
            }
            case DURING -> {
                ClientCast c = ClientCastManager.local();
                if (c == null) {
                    if (++wait > 30) { // the cast never started or already ended
                        phase = Phase.AFTER;
                        wait = 0;
                    }
                } else {
                    wait = 0;
                }
            }
            case AFTER -> {
                if (++wait == 10) {
                    boolean ok = ClientCastManager.local() == null
                            && !CinematicCamera.active()
                            && mc.options.getCameraType() == startCamera();
                    log("restored=" + ok + " camera=" + mc.options.getCameraType());
                }
                if (wait > 40) { // cooldown (108 ticks) has not elapsed yet; clear it so repeats are immediate
                    mc.player.getCooldowns().removeCooldown(dev.fairytail.registry.ModItems.KEYS.get(current()).get());
                    if (++rep >= REPEAT) {
                        rep = 0;
                        keyIdx++;
                        if (!"all".equalsIgnoreCase(KEYSPEC) || keyIdx >= ZodiacKey.values().length) {
                            phase = Phase.DONE;
                            log("AUTOTEST_DONE");
                            mc.stop();
                            return;
                        }
                        phase = Phase.GIVE;
                    } else {
                        phase = Phase.CAST;
                    }
                    wait = 0;
                }
            }
            case DONE -> {}
        }
    }

    private static CameraType startCamera() {
        return switch (CAMERA) {
            case "back" -> CameraType.THIRD_PERSON_BACK;
            case "front" -> CameraType.THIRD_PERSON_FRONT;
            default -> CameraType.FIRST_PERSON;
        };
    }

    /** Called after each rendered frame: screenshot when the local cast crosses the next requested tick. */
    public static void frame() {
        if (!enabled() || phase != Phase.DURING) return;
        Minecraft mc = Minecraft.getInstance();
        ClientCast c = ClientCastManager.local();
        if (c == null || mc.player == null || nextShot >= SHOTS.length) return;
        float t = c.time(mc.player, mc.getTimer().getGameTimeDeltaPartialTick(false));
        if (t >= SHOTS[nextShot]) {
            String name = String.format("%s_r%d_%02d_t%05.1f.png", current().id, rep + 1, nextShot, t);
            Screenshot.grab(mc.gameDirectory, name, mc.getMainRenderTarget(), msg -> {});
            log("shot " + name + " pos=" + String.format("%.2f,%.2f,%.2f", mc.gameRenderer.getMainCamera().getPosition().x, mc.gameRenderer.getMainCamera().getPosition().y, mc.gameRenderer.getMainCamera().getPosition().z)
                    + " keyAxis=" + String.format("%.2f,%.2f,%.2f", c.axis.x, c.axis.y, c.axis.z)
                    + " facingYaw=" + String.format("%.1f", c.facingYaw)
                    + " fwd=" + String.format("%.2f,%.2f", -Math.sin(Math.toRadians(c.facingYaw)), Math.cos(Math.toRadians(c.facingYaw))));
            nextShot++;
        }
    }

    private static void log(String s) {
        FairyTailMod.LOGGER.info("[autotest] {}", s);
    }

    private AutoTest() {}
}
