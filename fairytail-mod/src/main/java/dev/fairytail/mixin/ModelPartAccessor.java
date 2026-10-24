package dev.fairytail.mixin;

import net.minecraft.client.model.geom.ModelPart;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Mutable;
import org.spongepowered.asm.mixin.gen.Accessor;

import java.util.List;
import java.util.Map;

@Mixin(ModelPart.class)
public interface ModelPartAccessor {
    @Mutable
    @Accessor("cubes")
    void fairytail$setCubes(List<ModelPart.Cube> cubes);

    @Accessor("children")
    Map<String, ModelPart> fairytail$getChildren();

    @Mutable
    @Accessor("children")
    void fairytail$setChildren(Map<String, ModelPart> children);
}
