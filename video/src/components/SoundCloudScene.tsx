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

// SoundCloud orange
const scOrange = "#FF5500";
const scOrangeLight = "#FF7733";

export const SoundCloudScene: React.FC<Props> = ({ colors }) => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  const titleProgress = spring({
    frame,
    fps,
    config: { damping: 18, stiffness: 90 },
  });

  const cardProgress = spring({
    frame: frame - fps * 0.3,
    fps,
    config: { damping: 15, stiffness: 70 },
  });

  // Animated sync icon rotation - spread over 6 seconds
  const syncRotation = interpolate(
    frame,
    [fps * 1.2, fps * 2.5, fps * 3.8, fps * 5.0],
    [0, 360, 720, 1080],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.inOut(Easing.quad) }
  );

  // Track items animation - spread over 6 seconds
  const trackItems = [
    { title: "Deep House Mix Vol. 3", artist: "DJ Shadow", delay: 2.0 },
    { title: "Sunset Groove", artist: "Beach Collective", delay: 2.6 },
    { title: "Bass Culture", artist: "Bass Master", delay: 3.2 },
  ];

  const isWide = width > height;
  const glowPulse = Math.sin(frame * 0.1) * 0.3 + 0.7;

  // Flowing lines animation
  const flowOffset = (frame * 2) % 200;

  return (
    <AbsoluteFill
      style={{
        fontFamily,
        justifyContent: "center",
        alignItems: "center",
        background: `radial-gradient(ellipse at 60% 40%, ${colors.surface} 0%, ${colors.black} 70%)`,
        padding: isWide ? 60 : 40,
      }}
    >
      {/* Animated connection lines in background */}
      <svg
        style={{
          position: "absolute",
          width: "100%",
          height: "100%",
          opacity: 0.15,
        }}
      >
        <defs>
          <linearGradient id="lineGrad" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor={scOrange} stopOpacity="0" />
            <stop offset="50%" stopColor={scOrange} stopOpacity="1" />
            <stop offset="100%" stopColor={colors.amber} stopOpacity="0" />
          </linearGradient>
        </defs>
        {[0, 1, 2].map((i) => (
          <line
            key={i}
            x1={`${20 + flowOffset + i * 30}%`}
            y1={`${30 + i * 15}%`}
            x2={`${50 + flowOffset + i * 30}%`}
            y2={`${50 + i * 10}%`}
            stroke="url(#lineGrad)"
            strokeWidth="2"
          />
        ))}
      </svg>

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          maxWidth: isWide ? width * 0.8 : width * 0.95,
        }}
      >
        {/* Title */}
        <div
          style={{
            opacity: interpolate(titleProgress, [0, 1], [0, 1]),
            transform: `translateY(${interpolate(titleProgress, [0, 1], [-30, 0])}px)`,
            marginBottom: 16,
          }}
        >
          <span
            style={{
              fontSize: width * 0.016,
              color: scOrange,
              fontWeight: 600,
              textTransform: "uppercase",
              letterSpacing: "0.15em",
              background: `${scOrange}15`,
              padding: "10px 20px",
              borderRadius: 100,
              border: `1px solid ${scOrange}40`,
            }}
          >
            Auto-Sync Feature
          </span>
        </div>

        <h2
          style={{
            fontSize: width * 0.038,
            fontWeight: 800,
            color: colors.cream,
            marginBottom: 12,
            textAlign: "center",
            opacity: interpolate(titleProgress, [0, 1], [0, 1]),
            transform: `translateY(${interpolate(titleProgress, [0, 1], [-20, 0])}px)`,
          }}
        >
          Your{" "}
          <span style={{ color: scOrange }}>SoundCloud</span>{" "}
          likes,
        </h2>
        <h2
          style={{
            fontSize: width * 0.038,
            fontWeight: 800,
            color: colors.cream,
            marginBottom: 40,
            textAlign: "center",
            opacity: interpolate(titleProgress, [0, 1], [0, 1]),
            transform: `translateY(${interpolate(titleProgress, [0, 1], [-20, 0])}px)`,
          }}
        >
          automatically{" "}
          <span
            style={{
              background: `linear-gradient(90deg, ${colors.amber}, ${colors.cyan})`,
              WebkitBackgroundClip: "text",
              WebkitTextFillColor: "transparent",
            }}
          >
            downloaded
          </span>
        </h2>

        {/* Visual representation */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: isWide ? 60 : 30,
            opacity: interpolate(cardProgress, [0, 1], [0, 1]),
            transform: `scale(${interpolate(cardProgress, [0, 1], [0.9, 1])})`,
          }}
        >
          {/* SoundCloud side */}
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              gap: 16,
            }}
          >
            <div
              style={{
                width: isWide ? 100 : 70,
                height: isWide ? 100 : 70,
                borderRadius: 20,
                background: `linear-gradient(135deg, ${scOrange}, ${scOrangeLight})`,
                display: "flex",
                justifyContent: "center",
                alignItems: "center",
                boxShadow: `0 0 ${40 * glowPulse}px ${scOrange}50`,
                fontSize: isWide ? 48 : 32,
              }}
            >
              ☁️
            </div>
            <span
              style={{
                color: colors.muted,
                fontSize: width * 0.014,
                fontWeight: 600,
              }}
            >
              SoundCloud Likes
            </span>
          </div>

          {/* Sync arrows */}
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              gap: 8,
            }}
          >
            <div
              style={{
                fontSize: isWide ? 36 : 24,
                transform: `rotate(${syncRotation}deg)`,
                color: colors.cyan,
              }}
            >
              ⟳
            </div>
            <span
              style={{
                color: colors.subtle,
                fontSize: width * 0.011,
                textTransform: "uppercase",
                letterSpacing: "0.1em",
              }}
            >
              Every 24h
            </span>
          </div>

          {/* CrateDrop side */}
          <div
            style={{
              background: colors.surface,
              borderRadius: 16,
              border: `1px solid ${colors.border}`,
              padding: isWide ? 20 : 12,
              minWidth: isWide ? 320 : 200,
              boxShadow: `0 8px 32px rgba(0,0,0,0.4), 0 0 ${20 * glowPulse}px ${colors.amber}15`,
            }}
          >
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                marginBottom: 16,
                paddingBottom: 12,
                borderBottom: `1px solid ${colors.border}`,
              }}
            >
              <span style={{ fontSize: isWide ? 20 : 14 }}>📦</span>
              <span
                style={{
                  color: colors.cream,
                  fontWeight: 700,
                  fontSize: width * 0.014,
                }}
              >
                SoundCloud Likes
              </span>
            </div>

            {trackItems.map((track, i) => {
              const trackProgress = spring({
                frame: frame - track.delay * fps,
                fps,
                config: { damping: 15, stiffness: 100 },
              });

              return (
                <div
                  key={i}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 12,
                    padding: "10px 0",
                    borderBottom: i < trackItems.length - 1 ? `1px solid ${colors.border}` : "none",
                    opacity: interpolate(trackProgress, [0, 1], [0, 1]),
                    transform: `translateX(${interpolate(trackProgress, [0, 1], [20, 0])}px)`,
                  }}
                >
                  <div
                    style={{
                      width: isWide ? 36 : 28,
                      height: isWide ? 36 : 28,
                      borderRadius: 6,
                      background: `linear-gradient(135deg, ${colors.amber}80, ${colors.amberDark})`,
                      display: "flex",
                      justifyContent: "center",
                      alignItems: "center",
                      fontSize: isWide ? 14 : 10,
                      color: colors.black,
                      fontWeight: 700,
                    }}
                  >
                    ♪
                  </div>
                  <div style={{ flex: 1 }}>
                    <div
                      style={{
                        color: colors.cream,
                        fontSize: width * 0.012,
                        fontWeight: 600,
                        whiteSpace: "nowrap",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                      }}
                    >
                      {track.title}
                    </div>
                    <div
                      style={{
                        color: colors.subtle,
                        fontSize: width * 0.01,
                      }}
                    >
                      {track.artist}
                    </div>
                  </div>
                  <div
                    style={{
                      color: "#4ADE80",
                      fontSize: width * 0.012,
                      opacity: interpolate(trackProgress, [0, 1], [0, 1]),
                    }}
                  >
                    ✓
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Bottom tagline */}
        <p
          style={{
            marginTop: 40,
            color: colors.muted,
            fontSize: width * 0.016,
            textAlign: "center",
            opacity: interpolate(frame, [fps * 4, fps * 4.5], [0, 1], {
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            }),
          }}
        >
          320kbps MP3 • Automatic duplicate detection • Zero effort
        </p>
      </div>
    </AbsoluteFill>
  );
};
