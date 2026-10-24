package dev.fairytail.client.model;

import net.minecraft.client.model.geom.ModelPart;

/** Implemented (via mixin) by PlayerModel: exposes the extra elbow-joint parts we split the arms into. */
public interface ElbowModel {
    ModelPart fairytail$forearm(boolean left);

    ModelPart fairytail$forearmSleeve(boolean left);
}
