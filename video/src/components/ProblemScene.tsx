import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { loadFont } from "@remotion/google-fonts/Inter";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "600", "700"],
  subsets: ["latin"],
});

const problems = [
  { icon: "☁️", text: "Streaming services disappear tracks" },
  { icon: "💸", text: "Monthly subscriptions add up" },
  { icon: "🔒", text: "No control over your library" },
  { icon: "📡", text: "Need internet for everything" },
];

const ProblemItem: React.FC<{
  icon: string;
  text: string;
  index: number;
}> = ({ icon, text, index }) => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  const delay = index * 0.3;
  const entryProgress = spring({
    frame: frame - delay * fps,
    fps,
    config: { damping: 15, stiffness: 100 },
  });

  const x = interpolate(entryProgress, [0, 1], [-100, 0]);
  const opacity = interpolate(entryProgress, [0, 1], [0, 1]);

  const strikeProgress = interpolate(
    frame,
    [(delay + 0.8) * fps, (delay + 1.1) * fps],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 24,
        opacity,
        transform: `translateX(${x}px)`,
        marginBottom: 28,
        position: "relative",
      }}
    >
      <div
        style={{
          fontSize: width * 0.035,
          width: 80,
          height: 80,
          background: "rgba(239, 68, 68, 0.15)",
          borderRadius: 16,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          border: "2px solid rgba(239, 68, 68, 0.3)",
        }}
      >
        {icon}
      </div>
      <p
        style={{
          fontSize: width * 0.026,
          color: "#e0e0e0",
          fontWeight: 600,
          position: "relative",
        }}
      >
        {text}
        <span
          style={{
            position: "absolute",
            left: 0,
            top: "50%",
            width: `${strikeProgress * 100}%`,
            height: 3,
            background: "#ef4444",
            transform: "translateY(-50%)",
          }}
        />
      </p>
    </div>
  );
};

export const ProblemScene: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  const titleOpacity = interpolate(frame, [0, fps * 0.3], [0, 1], {
    extrapolateRight: "clamp",
  });

  const titleY = interpolate(frame, [0, fps * 0.3], [-20, 0], {
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        padding: "80px 120px",
      }}
    >
      <h2
        style={{
          fontSize: width * 0.04,
          fontWeight: 700,
          color: "#ef4444",
          marginBottom: 60,
          opacity: titleOpacity,
          transform: `translateY(${titleY}px)`,
        }}
      >
        Tired of this?
      </h2>

      <div style={{ display: "flex", flexDirection: "column" }}>
        {problems.map((problem, i) => (
          <ProblemItem key={i} icon={problem.icon} text={problem.text} index={i} />
        ))}
      </div>
    </AbsoluteFill>
  );
};
