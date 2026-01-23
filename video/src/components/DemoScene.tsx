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
  weights: ["400", "500", "600", "700"],
  subsets: ["latin"],
});

type Props = {
  colors: CrateColors;
};

export const DemoScene: React.FC<Props> = ({ colors }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const mockupProgress = spring({
    frame,
    fps,
    config: { damping: 18, stiffness: 70 },
  });

  const mockupY = interpolate(mockupProgress, [0, 1], [40, 0]);
  const mockupOpacity = interpolate(mockupProgress, [0, 1], [0, 1]);

  const glowPulse = Math.sin(frame * 0.08) * 0.3 + 0.7;

  const isWide = width > height;
  const mockupWidth = isWide ? width * 0.72 : width * 0.92;

  // VU meter animation
  const vuLevel = Math.abs(Math.sin(frame * 0.15)) * 0.6 + 0.3;

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
        background: `radial-gradient(ellipse at 50% 60%, ${colors.surface} 0%, ${colors.black} 70%)`,
      }}
    >
      {/* Floating label */}
      <div
        style={{
          position: "absolute",
          top: isWide ? 50 : 30,
          left: "50%",
          transform: "translateX(-50%)",
          opacity: interpolate(frame, [fps * 0.4, fps * 0.8], [0, 1], {
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          }),
        }}
      >
        <span
          style={{
            fontSize: width * 0.015,
            color: colors.amber,
            fontWeight: 600,
            textTransform: "uppercase",
            letterSpacing: "0.15em",
            background: `${colors.amber}15`,
            padding: "10px 24px",
            borderRadius: 100,
            border: `1px solid ${colors.amber}40`,
          }}
        >
          Clean Interface
        </span>
      </div>

      {/* Browser mockup */}
      <div
        style={{
          width: mockupWidth,
          transform: `translateY(${mockupY}px)`,
          opacity: mockupOpacity,
          boxShadow: `0 40px 100px rgba(0,0,0,0.6), 0 0 ${60 * glowPulse}px ${colors.amber}10`,
          borderRadius: 12,
          overflow: "hidden",
          border: `1px solid ${colors.border}`,
        }}
      >
        {/* Browser chrome */}
        <div
          style={{
            height: 40,
            background: colors.elevated,
            display: "flex",
            alignItems: "center",
            padding: "0 14px",
            gap: 8,
            borderBottom: `1px solid ${colors.border}`,
          }}
        >
          <div style={{ width: 12, height: 12, borderRadius: "50%", background: "#ff5f57" }} />
          <div style={{ width: 12, height: 12, borderRadius: "50%", background: "#febc2e" }} />
          <div style={{ width: 12, height: 12, borderRadius: "50%", background: "#28c840" }} />
          <div
            style={{
              marginLeft: 12,
              flex: 1,
              height: 26,
              background: colors.surface,
              borderRadius: 6,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              border: `1px solid ${colors.border}`,
            }}
          >
            <span style={{ color: colors.subtle, fontSize: 12 }}>
              cratedrop.local
            </span>
          </div>
        </div>

        {/* App UI mockup */}
        <div
          style={{
            height: isWide ? height * 0.52 : height * 0.42,
            background: colors.black,
            position: "relative",
            display: "flex",
          }}
        >
          {/* Sidebar */}
          <div
            style={{
              width: "16%",
              background: colors.surface,
              borderRight: `1px solid ${colors.border}`,
              padding: 16,
              display: "flex",
              flexDirection: "column",
            }}
          >
            <div
              style={{
                color: colors.cream,
                fontSize: 14,
                fontWeight: 700,
                marginBottom: 24,
                display: "flex",
                alignItems: "center",
                gap: 8,
              }}
            >
              <span style={{ color: colors.amber }}>◉</span> CrateDrop
            </div>
            {["Library", "Crates", "Upload", "Community"].map((item, i) => (
              <div
                key={item}
                style={{
                  padding: "10px 12px",
                  borderRadius: 8,
                  marginBottom: 4,
                  background: i === 0 ? `${colors.amber}20` : "transparent",
                  color: i === 0 ? colors.cream : colors.muted,
                  fontSize: 13,
                  fontWeight: 500,
                  border: i === 0 ? `1px solid ${colors.amber}30` : "none",
                }}
              >
                {item}
              </div>
            ))}
          </div>

          {/* Main content */}
          <div style={{ flex: 1, padding: 20 }}>
            <h3
              style={{
                color: colors.cream,
                fontSize: 20,
                fontWeight: 700,
                marginBottom: 20,
              }}
            >
              Your Library
            </h3>

            {/* Track list */}
            {[
              { title: "Deep House Mix Vol. 3", artist: "DJ Shadow", duration: "6:42" },
              { title: "Techno Warehouse Set", artist: "Producer X", duration: "8:15" },
              { title: "Summer Vibes 2024", artist: "Beach Collective", duration: "5:30" },
              { title: "Underground Bass", artist: "Bass Master", duration: "7:22" },
            ].map((track, i) => {
              const trackDelay = 0.4 + i * 0.12;
              const trackProgress = spring({
                frame: frame - trackDelay * fps,
                fps,
                config: { damping: 18, stiffness: 100 },
              });

              return (
                <div
                  key={i}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    padding: "12px 14px",
                    background: i === 0 ? colors.surface : "transparent",
                    borderRadius: 8,
                    marginBottom: 6,
                    opacity: interpolate(trackProgress, [0, 1], [0, 1]),
                    transform: `translateX(${interpolate(trackProgress, [0, 1], [15, 0])}px)`,
                    border: i === 0 ? `1px solid ${colors.amber}30` : `1px solid transparent`,
                  }}
                >
                  <div
                    style={{
                      width: 36,
                      height: 36,
                      background: i === 0
                        ? `linear-gradient(135deg, ${colors.amber}, ${colors.amberDark})`
                        : colors.elevated,
                      borderRadius: 6,
                      display: "flex",
                      justifyContent: "center",
                      alignItems: "center",
                      marginRight: 14,
                      fontSize: 14,
                      color: i === 0 ? colors.black : colors.muted,
                    }}
                  >
                    {i === 0 ? "▶" : "♪"}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ color: colors.cream, fontSize: 14, fontWeight: 600 }}>
                      {track.title}
                    </div>
                    <div style={{ color: colors.subtle, fontSize: 12 }}>{track.artist}</div>
                  </div>
                  <div style={{ color: colors.subtle, fontSize: 13 }}>{track.duration}</div>
                </div>
              );
            })}
          </div>

          {/* Player bar */}
          <div
            style={{
              position: "absolute",
              bottom: 0,
              left: 0,
              right: 0,
              height: 64,
              background: colors.surface,
              borderTop: `1px solid ${colors.border}`,
              display: "flex",
              alignItems: "center",
              padding: "0 20px",
            }}
          >
            <div
              style={{
                width: 44,
                height: 44,
                background: `linear-gradient(135deg, ${colors.amber}, ${colors.amberDark})`,
                borderRadius: 8,
                marginRight: 14,
              }}
            />
            <div style={{ flex: 1 }}>
              <div style={{ color: colors.cream, fontSize: 13, fontWeight: 600 }}>
                Deep House Mix Vol. 3
              </div>
              <div style={{ color: colors.subtle, fontSize: 11 }}>DJ Shadow</div>
            </div>

            {/* VU Meter */}
            <div
              style={{
                display: "flex",
                alignItems: "flex-end",
                gap: 3,
                height: 30,
                marginRight: 24,
              }}
            >
              {[0.6, 0.8, 1, 0.7, 0.9, 0.5, 0.8].map((base, i) => {
                const barHeight = base * vuLevel * 30;
                const isHigh = barHeight > 20;
                return (
                  <div
                    key={i}
                    style={{
                      width: 4,
                      height: barHeight,
                      borderRadius: 2,
                      background: isHigh
                        ? `linear-gradient(to top, ${colors.amber}, ${colors.cyan})`
                        : colors.amber,
                    }}
                  />
                );
              })}
            </div>

            <div style={{ display: "flex", gap: 16, alignItems: "center" }}>
              <span style={{ color: colors.muted, fontSize: 16 }}>⏮</span>
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: "50%",
                  background: colors.amber,
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                  color: colors.black,
                  fontSize: 14,
                  boxShadow: `0 0 20px ${colors.amber}40`,
                }}
              >
                ▶
              </div>
              <span style={{ color: colors.muted, fontSize: 16 }}>⏭</span>
            </div>
          </div>
        </div>
      </div>
    </AbsoluteFill>
  );
};
