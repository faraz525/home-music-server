import {
  AbsoluteFill,
  interpolate,
  spring,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";
import { loadFont } from "@remotion/google-fonts/Inter";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "700", "900"],
  subsets: ["latin"],
});

type Props = {
  brandColor: string;
};

export const IntroScene: React.FC<Props> = ({ brandColor }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const logoScale = spring({
    frame,
    fps,
    config: { damping: 12, stiffness: 100 },
  });

  const logoRotation = interpolate(frame, [0, fps * 0.5], [-180, 0], {
    extrapolateRight: "clamp",
  });

  const textOpacity = interpolate(frame, [fps * 0.5, fps * 1], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const textY = interpolate(frame, [fps * 0.5, fps * 1], [30, 0], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const taglineOpacity = interpolate(frame, [fps * 1.2, fps * 1.7], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const glowPulse = Math.sin(frame * 0.1) * 0.3 + 0.7;

  return (
    <AbsoluteFill
      style={{
        justifyContent: "center",
        alignItems: "center",
        fontFamily,
      }}
    >
      {/* Animated background particles */}
      <div
        style={{
          position: "absolute",
          width: "100%",
          height: "100%",
          overflow: "hidden",
        }}
      >
        {Array.from({ length: 20 }).map((_, i) => {
          const x = (i * 137.5) % 100;
          const y = (i * 73.7) % 100;
          const delay = i * 0.1;
          const particleOpacity = interpolate(
            frame,
            [delay * fps, (delay + 0.5) * fps],
            [0, 0.3],
            { extrapolateRight: "clamp" }
          );
          return (
            <div
              key={i}
              style={{
                position: "absolute",
                left: `${x}%`,
                top: `${y}%`,
                width: 4 + (i % 3) * 2,
                height: 4 + (i % 3) * 2,
                borderRadius: "50%",
                background: brandColor,
                opacity: particleOpacity * glowPulse,
                filter: "blur(1px)",
              }}
            />
          );
        })}
      </div>

      {/* Logo container */}
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          transform: `scale(${logoScale})`,
        }}
      >
        {/* Vinyl record icon */}
        <div
          style={{
            width: width * 0.15,
            height: width * 0.15,
            borderRadius: "50%",
            background: `conic-gradient(from ${logoRotation}deg, #1a1a2e, #2d2d44, #1a1a2e)`,
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            boxShadow: `0 0 ${60 * glowPulse}px ${brandColor}40`,
            border: `3px solid ${brandColor}`,
          }}
        >
          {/* Inner ring */}
          <div
            style={{
              width: "70%",
              height: "70%",
              borderRadius: "50%",
              background: `radial-gradient(circle at 30% 30%, #3d3d5c, #1a1a2e)`,
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              border: `2px solid ${brandColor}60`,
            }}
          >
            {/* Center */}
            <div
              style={{
                width: "30%",
                height: "30%",
                borderRadius: "50%",
                background: brandColor,
                boxShadow: `0 0 20px ${brandColor}`,
              }}
            />
          </div>
        </div>

        {/* Brand name */}
        <h1
          style={{
            fontSize: width * 0.08,
            fontWeight: 900,
            color: "white",
            marginTop: 40,
            opacity: textOpacity,
            transform: `translateY(${textY}px)`,
            letterSpacing: "-0.02em",
          }}
        >
          Crate<span style={{ color: brandColor }}>Drop</span>
        </h1>

        {/* Tagline */}
        <p
          style={{
            fontSize: width * 0.022,
            color: "#a0a0b0",
            marginTop: 16,
            opacity: taglineOpacity,
            fontWeight: 400,
            letterSpacing: "0.1em",
            textTransform: "uppercase",
          }}
        >
          Your Music. Your Server. Your Rules.
        </p>
      </div>
    </AbsoluteFill>
  );
};
