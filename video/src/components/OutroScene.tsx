import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { loadFont } from "@remotion/google-fonts/Inter";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "600", "700", "900"],
  subsets: ["latin"],
});

type Props = {
  brandColor: string;
  accentColor: string;
};

export const OutroScene: React.FC<Props> = ({ brandColor, accentColor }) => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  const ctaScale = spring({
    frame,
    fps,
    config: { damping: 10, stiffness: 80 },
  });

  const ctaY = interpolate(ctaScale, [0, 1], [40, 0]);

  const buttonScale = spring({
    frame: frame - fps * 0.5,
    fps,
    config: { damping: 8, stiffness: 100 },
  });

  const glowPulse = Math.sin(frame * 0.15) * 0.4 + 0.6;

  const taglineOpacity = interpolate(frame, [fps * 0.8, fps * 1.2], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
        background: `radial-gradient(ellipse at center, ${brandColor}10 0%, transparent 70%)`,
      }}
    >
      {/* Animated rings */}
      {[1, 2, 3].map((ring) => {
        const ringProgress = interpolate(
          frame,
          [(ring - 1) * 0.2 * fps, (ring - 1) * 0.2 * fps + fps],
          [0, 1],
          { extrapolateRight: "clamp" }
        );
        const ringScale = interpolate(ringProgress, [0, 1], [0.5, 1 + ring * 0.2]);
        const ringOpacity = interpolate(ringProgress, [0, 0.3, 1], [0, 0.3, 0]);

        return (
          <div
            key={ring}
            style={{
              position: "absolute",
              width: width * 0.5,
              height: width * 0.5,
              borderRadius: "50%",
              border: `2px solid ${brandColor}`,
              opacity: ringOpacity,
              transform: `scale(${ringScale})`,
            }}
          />
        );
      })}

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          transform: `scale(${ctaScale}) translateY(${ctaY}px)`,
        }}
      >
        <h2
          style={{
            fontSize: width * 0.055,
            fontWeight: 900,
            color: "white",
            marginBottom: 24,
            textAlign: "center",
            lineHeight: 1.2,
          }}
        >
          Take Control of
          <br />
          <span
            style={{
              background: `linear-gradient(90deg, ${brandColor}, ${accentColor})`,
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
            }}
          >
            Your Music
          </span>
        </h2>

        <p
          style={{
            fontSize: width * 0.02,
            color: "#a0a0b0",
            marginBottom: 40,
            opacity: taglineOpacity,
            textAlign: "center",
          }}
        >
          Open source. Self-hosted. Forever yours.
        </p>

        {/* CTA Button */}
        <div
          style={{
            transform: `scale(${buttonScale})`,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <div
            style={{
              background: `linear-gradient(135deg, ${brandColor}, ${accentColor})`,
              padding: "20px 48px",
              borderRadius: 100,
              boxShadow: `0 0 ${50 * glowPulse}px ${brandColor}60`,
              display: "flex",
              alignItems: "center",
              gap: 12,
            }}
          >
            <span
              style={{
                fontSize: width * 0.022,
                fontWeight: 700,
                color: "white",
                letterSpacing: "-0.01em",
              }}
            >
              Get Started Free
            </span>
            <span style={{ fontSize: width * 0.02 }}>→</span>
          </div>

          <div
            style={{
              marginTop: 24,
              display: "flex",
              alignItems: "center",
              gap: 8,
              opacity: taglineOpacity,
            }}
          >
            <span style={{ fontSize: 20 }}>⭐</span>
            <span
              style={{
                color: "#6b6b80",
                fontSize: width * 0.014,
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
