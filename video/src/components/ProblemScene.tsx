import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
  Easing,
} from "remotion";
import { loadFont } from "@remotion/google-fonts/DmSans";
import type { CrateColors } from "../CrateDropPromo";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "600", "700"],
  subsets: ["latin"],
});

type Props = {
  colors: CrateColors;
};

const problems = [
  { icon: "☁️", text: "Streaming services remove your tracks" },
  { icon: "💸", text: "Monthly subscriptions drain your wallet" },
  { icon: "🔒", text: "Zero control over your own library" },
  { icon: "📡", text: "No internet = no music" },
];

const ProblemItem: React.FC<{
  icon: string;
  text: string;
  index: number;
  colors: CrateColors;
}> = ({ icon, text, index, colors }) => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  const delay = index * 0.9;
  const entryProgress = spring({
    frame: frame - delay * fps,
    fps,
    config: { damping: 18, stiffness: 120 },
  });

  const x = interpolate(entryProgress, [0, 1], [-60, 0]);
  const opacity = interpolate(entryProgress, [0, 1], [0, 1], {
    easing: Easing.out(Easing.quad),
  });

  const strikeProgress = interpolate(
    frame,
    [(delay + 1.8) * fps, (delay + 2.3) * fps],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.quad) }
  );

  const dangerColor = "#FF6B6B";

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 28,
        opacity,
        transform: `translateX(${x}px)`,
        marginBottom: 32,
      }}
    >
      <div
        style={{
          fontSize: width * 0.032,
          width: 72,
          height: 72,
          background: `${dangerColor}15`,
          borderRadius: 18,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          border: `2px solid ${dangerColor}30`,
          flexShrink: 0,
        }}
      >
        {icon}
      </div>
      <p
        style={{
          fontSize: width * 0.024,
          color: colors.cream,
          fontWeight: 600,
          position: "relative",
        }}
      >
        {text}
        <span
          style={{
            position: "absolute",
            left: -4,
            top: "50%",
            width: `calc(${strikeProgress * 100}% + 8px)`,
            height: 3,
            background: `linear-gradient(90deg, ${dangerColor}, ${dangerColor}80)`,
            transform: "translateY(-50%)",
            borderRadius: 2,
          }}
        />
      </p>
    </div>
  );
};

export const ProblemScene: React.FC<Props> = ({ colors }) => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  const titleProgress = spring({
    frame,
    fps,
    config: { damping: 20, stiffness: 100 },
  });

  const titleOpacity = interpolate(titleProgress, [0, 1], [0, 1]);
  const titleY = interpolate(titleProgress, [0, 1], [-30, 0]);

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        padding: "80px 140px",
        background: `radial-gradient(ellipse at 30% 30%, ${colors.surface} 0%, ${colors.black} 60%)`,
      }}
    >
      <h2
        style={{
          fontSize: width * 0.042,
          fontWeight: 700,
          color: "#FF6B6B",
          marginBottom: 56,
          opacity: titleOpacity,
          transform: `translateY(${titleY}px)`,
          textShadow: "0 0 40px rgba(255, 107, 107, 0.3)",
        }}
      >
        Sound familiar?
      </h2>

      <div style={{ display: "flex", flexDirection: "column" }}>
        {problems.map((problem, i) => (
          <ProblemItem
            key={i}
            icon={problem.icon}
            text={problem.text}
            index={i}
            colors={colors}
          />
        ))}
      </div>
    </AbsoluteFill>
  );
};
