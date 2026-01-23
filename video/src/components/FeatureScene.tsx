import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
  Sequence,
} from "remotion";
import { loadFont } from "@remotion/google-fonts/Inter";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "600", "700", "800"],
  subsets: ["latin"],
});

type Props = {
  brandColor: string;
  accentColor: string;
};

const features = [
  {
    icon: "🎵",
    title: "Upload Any Format",
    description: "WAV, FLAC, AIFF, MP3 — all your originals",
  },
  {
    icon: "📦",
    title: "Organize in Crates",
    description: "Create playlists for every gig",
  },
  {
    icon: "🔊",
    title: "Stream Anywhere",
    description: "Access from any device, any browser",
  },
  {
    icon: "🏠",
    title: "Self-Hosted",
    description: "Runs on your own hardware",
  },
];

const FeatureCard: React.FC<{
  icon: string;
  title: string;
  description: string;
  index: number;
  brandColor: string;
  accentColor: string;
}> = ({ icon, title, description, index, brandColor, accentColor }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const delay = index * 0.25;
  const entryProgress = spring({
    frame: frame - delay * fps,
    fps,
    config: { damping: 12, stiffness: 80 },
  });

  const scale = interpolate(entryProgress, [0, 1], [0.8, 1]);
  const opacity = interpolate(entryProgress, [0, 1], [0, 1]);
  const y = interpolate(entryProgress, [0, 1], [40, 0]);

  const glowIntensity = interpolate(
    frame,
    [(delay + 0.5) * fps, (delay + 1) * fps],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" }
  );

  const isWide = width > height;
  const cardWidth = isWide ? width * 0.2 : width * 0.4;

  return (
    <div
      style={{
        width: cardWidth,
        padding: 32,
        background: `linear-gradient(135deg, rgba(255,255,255,0.05) 0%, rgba(255,255,255,0.02) 100%)`,
        borderRadius: 24,
        border: `1px solid rgba(255,255,255,0.1)`,
        opacity,
        transform: `scale(${scale}) translateY(${y}px)`,
        boxShadow: `0 0 ${40 * glowIntensity}px ${brandColor}20`,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        textAlign: "center",
      }}
    >
      <div
        style={{
          fontSize: width * 0.04,
          marginBottom: 20,
          background: `linear-gradient(135deg, ${brandColor}30, ${accentColor}20)`,
          width: width * 0.06,
          height: width * 0.06,
          borderRadius: 20,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
        }}
      >
        {icon}
      </div>
      <h3
        style={{
          fontSize: width * 0.02,
          fontWeight: 700,
          color: "white",
          marginBottom: 12,
        }}
      >
        {title}
      </h3>
      <p
        style={{
          fontSize: width * 0.014,
          color: "#a0a0b0",
          lineHeight: 1.5,
        }}
      >
        {description}
      </p>
    </div>
  );
};

export const FeatureScene: React.FC<Props> = ({ brandColor, accentColor }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const titleOpacity = interpolate(frame, [0, fps * 0.3], [0, 1], {
    extrapolateRight: "clamp",
  });
  const titleY = interpolate(frame, [0, fps * 0.3], [-30, 0], {
    extrapolateRight: "clamp",
  });

  const isWide = width > height;

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
        padding: isWide ? "60px 80px" : "40px",
      }}
    >
      <h2
        style={{
          fontSize: width * 0.045,
          fontWeight: 800,
          color: "white",
          marginBottom: 60,
          opacity: titleOpacity,
          transform: `translateY(${titleY}px)`,
          textAlign: "center",
        }}
      >
        Everything DJs{" "}
        <span
          style={{
            background: `linear-gradient(90deg, ${brandColor}, ${accentColor})`,
            WebkitBackgroundClip: "text",
            WebkitTextFillColor: "transparent",
          }}
        >
          actually need
        </span>
      </h2>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          gap: 24,
          justifyContent: "center",
        }}
      >
        {features.map((feature, i) => (
          <Sequence key={i} from={0} premountFor={fps}>
            <FeatureCard
              icon={feature.icon}
              title={feature.title}
              description={feature.description}
              index={i}
              brandColor={brandColor}
              accentColor={accentColor}
            />
          </Sequence>
        ))}
      </div>
    </AbsoluteFill>
  );
};
