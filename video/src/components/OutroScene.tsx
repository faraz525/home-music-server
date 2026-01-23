import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
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

export const OutroScene: React.FC<Props> = ({ colors }) => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  const contentProgress = spring({
    frame,
    fps,
    config: { damping: 16, stiffness: 80 },
  });

  const buttonProgress = spring({
    frame: frame - fps * 0.4,
    fps,
    config: { damping: 12, stiffness: 100 },
  });

  const glowPulse = Math.sin(frame * 0.12) * 0.4 + 0.6;

  const taglineOpacity = interpolate(frame, [fps * 0.6, fps * 1], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // Animated rings
  const rings = [1, 2, 3].map((ring) => {
    const ringProgress = interpolate(
      frame,
      [(ring - 1) * 0.15 * fps, (ring - 1) * 0.15 * fps + fps * 0.8],
      [0, 1],
      { extrapolateRight: "clamp" }
    );
    const ringScale = interpolate(ringProgress, [0, 1], [0.6, 1 + ring * 0.15]);
    const ringOpacity = interpolate(ringProgress, [0, 0.2, 1], [0, 0.25, 0]);
    return { scale: ringScale, opacity: ringOpacity };
  });

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
        background: `radial-gradient(ellipse at center, ${colors.amber}08 0%, ${colors.black} 60%)`,
      }}
    >
      {/* Animated rings */}
      {rings.map((ring, i) => (
        <div
          key={i}
          style={{
            position: "absolute",
            width: width * 0.4,
            height: width * 0.4,
            borderRadius: "50%",
            border: `2px solid ${colors.amber}`,
            opacity: ring.opacity,
            transform: `scale(${ring.scale})`,
          }}
        />
      ))}

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          opacity: interpolate(contentProgress, [0, 1], [0, 1]),
          transform: `translateY(${interpolate(contentProgress, [0, 1], [30, 0])}px)`,
        }}
      >
        <h2
          style={{
            fontSize: width * 0.048,
            fontWeight: 800,
            color: colors.cream,
            marginBottom: 12,
            textAlign: "center",
            lineHeight: 1.15,
          }}
        >
          Own Your
        </h2>
        <h2
          style={{
            fontSize: width * 0.048,
            fontWeight: 800,
            marginBottom: 28,
            textAlign: "center",
          }}
        >
          <span
            style={{
              background: `linear-gradient(90deg, ${colors.amber}, ${colors.cyan})`,
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
            }}
          >
            Music Library
          </span>
        </h2>

        <p
          style={{
            fontSize: width * 0.018,
            color: colors.muted,
            marginBottom: 36,
            opacity: taglineOpacity,
            textAlign: "center",
          }}
        >
          Open source • Self-hosted • Forever yours
        </p>

        {/* CTA Button */}
        <div
          style={{
            transform: `scale(${interpolate(buttonProgress, [0, 1], [0.9, 1])})`,
            opacity: interpolate(buttonProgress, [0, 1], [0, 1]),
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <div
            style={{
              background: `linear-gradient(135deg, ${colors.amber}, ${colors.amberLight})`,
              padding: "18px 44px",
              borderRadius: 100,
              boxShadow: `0 0 ${45 * glowPulse}px ${colors.amber}50, 0 8px 32px rgba(0,0,0,0.3)`,
              display: "flex",
              alignItems: "center",
              gap: 12,
            }}
          >
            <span
              style={{
                fontSize: width * 0.02,
                fontWeight: 700,
                color: colors.black,
                letterSpacing: "-0.01em",
              }}
            >
              Get Started Free
            </span>
            <span style={{ fontSize: width * 0.018, color: colors.black }}>→</span>
          </div>

          <div
            style={{
              marginTop: 20,
              display: "flex",
              alignItems: "center",
              gap: 8,
              opacity: taglineOpacity,
            }}
          >
            <span style={{ fontSize: 18 }}>⭐</span>
            <span
              style={{
                color: colors.subtle,
                fontSize: width * 0.013,
              }}
            >
              github.com/cratedrop
            </span>
          </div>
        </div>
      </div>
    </AbsoluteFill>
  );
};
