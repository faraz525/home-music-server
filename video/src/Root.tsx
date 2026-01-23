import { Composition, Folder } from "remotion";
import { CrateDropPromo } from "./CrateDropPromo";

// CrateDrop brand colors from tailwind.config.js
const crateColors = {
  black: "#0D0A14",
  surface: "#1A171F",
  elevated: "#252130",
  border: "#332E3C",
  amber: "#E5A000",
  amberLight: "#FFB82E",
  amberDark: "#B37D00",
  cyan: "#00D4FF",
  cyanDark: "#00A8CC",
  cream: "#F5F0E8",
  muted: "#9B95A3",
  subtle: "#6B6573",
};

export const RemotionRoot: React.FC = () => {
  return (
    <Folder name="CrateDrop">
      <Composition
        id="CrateDropPromo"
        component={CrateDropPromo}
        durationInFrames={540}
        fps={30}
        width={1920}
        height={1080}
        defaultProps={{
          colors: crateColors,
        }}
      />
      <Composition
        id="CrateDropPromo-Square"
        component={CrateDropPromo}
        durationInFrames={540}
        fps={30}
        width={1080}
        height={1080}
        defaultProps={{
          colors: crateColors,
        }}
      />
    </Folder>
  );
};
