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
  weights: ["400", "700", "800"],
  subsets: ["latin"],
});

type Props = {
  colors: CrateColors;
};

export const IntroScene: React.FC<Props> = ({ colors }) => {
  const frame = useCurrentFrame();
  const { fps, width } = useVideoConfig();

  // Smoother spring config
  const logoScale = spring({
    frame,
    fps,
    config: { damping: 15, stiffness: 80, mass: 1 },
  });

  const vinylRotation = interpolate(frame, [0, fps * 3], [0, 360], {
    extrapolateRight: "extend",
    easing: Easing.linear,
  });

  const textOpacity = interpolate(frame, [fps * 0.6, fps * 1.2], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
    easing: Easing.out(Easing.quad),
  });

  const textY = spring({
    frame: frame - fps * 0.6,
    fps,
    config: { damping: 20, stiffness: 100 },
  });

  const taglineOpacity = interpolate(frame, [fps * 1.4, fps * 2], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const taglineY = spring({
    frame: frame - fps * 1.4,
    fps,
    config: { damping: 20, stiffness: 80 },
  });

  const glowPulse = Math.sin(frame * 0.08) * 0.4 + 0.6;

  // Floating particles
  const particles = Array.from({ length: 30 }).map((_, i) => {
    const x = (i * 137.5 + frame * 0.2) % 110 - 5;
    const y = (i * 73.7 + frame * 0.15 * (i % 3 + 1)) % 110 - 5;
    const size = 2 + (i % 4) * 1.5;
    const delay = i * 0.05;
    const opacity = interpolate(
      frame,
      [delay * fps, (delay + 0.8) * fps],
      [0, 0.4],
      { extrapolateRight: "clamp" }
    ) * (0.3 + Math.sin(frame * 0.05 + i) * 0.2);

    return { x, y, size, opacity, isAmber: i % 3 === 0 };
  });

  return (
    <AbsoluteFill
      style={{
        justifyContent: "center",
        alignItems: "center",
        fontFamily,
        background: `radial-gradient(ellipse at 50% 50%, ${colors.surface} 0%, ${colors.black} 70%)`,
      }}
    >
      {/* Animated background particles */}
      <div style={{ position: "absolute", inset: 0, overflow: "hidden" }}>
        {particles.map((p, i) => (
          <div
            key={i}
            style={{
              position: "absolute",
              left: `${p.x}%`,
              top: `${p.y}%`,
              width: p.size,
              height: p.size,
              borderRadius: "50%",
              background: p.isAmber ? colors.amber : colors.cyan,
              opacity: p.opacity * glowPulse,
              filter: "blur(1px)",
              boxShadow: `0 0 ${p.size * 2}px ${p.isAmber ? colors.amber : colors.cyan}`,
            }}
          />
        ))}
      </div>

      {/* Logo container */}
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          transform: `scale(${interpolate(logoScale, [0, 1], [0.8, 1])})`,
          opacity: logoScale,
        }}
      >
        {/* Vinyl record icon */}
        <div
          style={{
            width: width * 0.14,
            height: width * 0.14,
            borderRadius: "50%",
            background: `conic-gradient(from ${vinylRotation}deg, ${colors.elevated}, ${colors.surface}, ${colors.elevated})`,
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            boxShadow: `0 0 ${80 * glowPulse}px ${colors.amber}30, inset 0 0 60px ${colors.black}`,
            border: `3px solid ${colors.amber}`,
            position: "relative",
          }}
        >
          {/* Grooves */}
          {[0.85, 0.75, 0.65].map((scale, i) => (
            <div
              key={i}
              style={{
                position: "absolute",
                width: `${scale * 100}%`,
                height: `${scale * 100}%`,
                borderRadius: "50%",
                border: `1px solid ${colors.border}`,
              }}
            />
          ))}
          {/* Inner label */}
          <div
            style={{
              width: "40%",
              height: "40%",
              borderRadius: "50%",
              background: `linear-gradient(135deg, ${colors.amber}, ${colors.amberDark})`,
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              boxShadow: `0 0 30px ${colors.amber}60`,
            }}
          >
            {/* Center hole */}
            <div
              style={{
                width: "20%",
                height: "20%",
                borderRadius: "50%",
                background: colors.black,
              }}
            />
          </div>
        </div>

        {/* Brand name */}
        <h1
          style={{
            fontSize: width * 0.075,
            fontWeight: 800,
            color: colors.cream,
            marginTop: 48,
            opacity: textOpacity,
            transform: `translateY(${interpolate(textY, [0, 1], [30, 0])}px)`,
            letterSpacing: "-0.03em",
            textShadow: `0 0 40px ${colors.amber}40`,
          }}
        >
          Crate<span style={{ color: colors.amber }}>Drop</span>
        </h1>

        {/* Tagline */}
        <p
          style={{
            fontSize: width * 0.02,
            color: colors.muted,
            marginTop: 20,
            opacity: taglineOpacity,
            transform: `translateY(${interpolate(taglineY, [0, 1], [20, 0])}px)`,
            fontWeight: 400,
            letterSpacing: "0.2em",
            textTransform: "uppercase",
          }}
        >
          Your Music • Your Server • Your Rules
        </p>
      </div>
    </AbsoluteFill>
  );
};
