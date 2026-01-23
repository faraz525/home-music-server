import { Composition, Folder } from "remotion";
import { CrateDropPromo } from "./CrateDropPromo";

export const RemotionRoot: React.FC = () => {
  return (
    <Folder name="CrateDrop">
      <Composition
        id="CrateDropPromo"
        component={CrateDropPromo}
        durationInFrames={450}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{
          brandColor: "#8B5CF6",
          accentColor: "#06B6D4",
        }}
      />
      <Composition
        id="CrateDropPromo-Square"
        component={CrateDropPromo}
        durationInFrames={450}
        fps={30}
        width={1080}
        height={1080}
        defaultProps={{
          brandColor: "#8B5CF6",
          accentColor: "#06B6D4",
        }}
      />
    </Folder>
  );
};
