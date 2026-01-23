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
  weights: ["400", "600", "700", "800"],
  subsets: ["latin"],
});

type Props = {
  colors: CrateColors;
};

const features = [
  {
    icon: "🎵",
    title: "Any Format",
    description: "WAV, FLAC, AIFF, MP3",
  },
  {
    icon: "📦",
    title: "Crate System",
    description: "Organize by gig or genre",
  },
  {
    icon: "🌐",
    title: "Stream Anywhere",
    description: "Any device, any browser",
  },
  {
    icon: "🏠",
    title: "Self-Hosted",
    description: "Your hardware, your data",
  },
];

const FeatureCard: React.FC<{
  icon: string;
  title: string;
  description: string;
  index: number;
  colors: CrateColors;
}> = ({ icon, title, description, index, colors }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const delay = index * 0.15;
  const entryProgress = spring({
    frame: frame - delay * fps,
    fps,
    config: { damping: 14, stiffness: 80 },
  });

  const scale = interpolate(entryProgress, [0, 1], [0.85, 1]);
  const opacity = interpolate(entryProgress, [0, 1], [0, 1]);
  const y = interpolate(entryProgress, [0, 1], [30, 0]);

  const glowIntensity = interpolate(
    frame,
    [(delay + 0.3) * fps, (delay + 0.8) * fps],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );

  const isWide = width > height;
  const cardWidth = isWide ? width * 0.19 : width * 0.42;

  return (
    <div
      style={{
        width: cardWidth,
        padding: 28,
        background: colors.surface,
        borderRadius: 20,
        border: `1px solid ${colors.border}`,
        opacity,
        transform: `scale(${scale}) translateY(${y}px)`,
        boxShadow: `0 0 ${30 * glowIntensity}px ${colors.amber}15, 0 8px 32px rgba(0,0,0,0.3)`,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        textAlign: "center",
      }}
    >
      <div
        style={{
          fontSize: width * 0.035,
          marginBottom: 16,
          background: `linear-gradient(135deg, ${colors.amber}25, ${colors.cyan}15)`,
          width: width * 0.055,
          height: width * 0.055,
          borderRadius: 16,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          border: `1px solid ${colors.border}`,
        }}
      >
        {icon}
      </div>
      <h3
        style={{
          fontSize: width * 0.018,
          fontWeight: 700,
          color: colors.cream,
          marginBottom: 8,
        }}
      >
        {title}
      </h3>
      <p
        style={{
          fontSize: width * 0.013,
          color: colors.muted,
          lineHeight: 1.4,
        }}
      >
        {description}
      </p>
    </div>
  );
};

export const FeatureScene: React.FC<Props> = ({ colors }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const titleProgress = spring({
    frame,
    fps,
    config: { damping: 18, stiffness: 90 },
  });

  const titleOpacity = interpolate(titleProgress, [0, 1], [0, 1]);
  const titleY = interpolate(titleProgress, [0, 1], [-25, 0]);

  const isWide = width > height;

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
        padding: isWide ? "50px 60px" : "40px",
        background: `radial-gradient(ellipse at 50% 30%, ${colors.surface} 0%, ${colors.black} 60%)`,
      }}
    >
      <h2
        style={{
          fontSize: width * 0.04,
          fontWeight: 800,
          color: colors.cream,
          marginBottom: 50,
          opacity: titleOpacity,
          transform: `translateY(${titleY}px)`,
          textAlign: "center",
        }}
      >
        Built for{" "}
        <span
          style={{
            background: `linear-gradient(90deg, ${colors.amber}, ${colors.amberLight})`,
            WebkitBackgroundClip: "text",
            WebkitTextFillColor: "transparent",
          }}
        >
          DJs
        </span>
      </h2>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 20,
          justifyContent: "center",
        }}
      >
        {features.map((feature, i) => (
          <FeatureCard
            key={i}
            icon={feature.icon}
            title={feature.title}
            description={feature.description}
            index={i}
            colors={colors}
          />
        ))}
      </div>
    </AbsoluteFill>
  );
};
