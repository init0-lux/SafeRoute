import React, { useState, useEffect, useRef } from 'react';
import {
    View,
    Text,
    StyleSheet,
    TouchableOpacity,
    Animated,
    Dimensions,
} from 'react-native';
import { router } from 'expo-router';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import PulseAnimation from '@/components/PulseAnimation';

const { width } = Dimensions.get('window');

type Phase = 'countdown' | 'calling';

export default function SOSScreen() {
    const [count, setCount] = useState(3);
    const [phase, setPhase] = useState<Phase>('countdown');
    const scaleAnim = useRef(new Animated.Value(0.5)).current;
    const opacityAnim = useRef(new Animated.Value(0)).current;

    // Animate each number
    const animateNumber = () => {
        scaleAnim.setValue(0.5);
        opacityAnim.setValue(0);

        Animated.parallel([
            Animated.sequence([
                Animated.timing(scaleAnim, {
                    toValue: 1.4,
                    duration: 400,
                    useNativeDriver: true,
                }),
                Animated.timing(scaleAnim, {
                    toValue: 1,
                    duration: 300,
                    useNativeDriver: true,
                }),
            ]),
            Animated.sequence([
                Animated.timing(opacityAnim, {
                    toValue: 1,
                    duration: 200,
                    useNativeDriver: true,
                }),
                Animated.delay(500),
                Animated.timing(opacityAnim, {
                    toValue: 0.3,
                    duration: 200,
                    useNativeDriver: true,
                }),
            ]),
        ]).start();
    };

    useEffect(() => {
        if (phase !== 'countdown') return;

        animateNumber();

        if (count === 0) {
            // After showing 0, switch to calling phase
            const timer = setTimeout(() => {
                setPhase('calling');
            }, 1000);
            return () => clearTimeout(timer);
        }

        const timer = setTimeout(() => {
            setCount((c) => c - 1);
        }, 1000);

        return () => clearTimeout(timer);
    }, [count, phase]);

    // Animate "calling" text entrance
    const callingScale = useRef(new Animated.Value(0.3)).current;
    const callingOpacity = useRef(new Animated.Value(0)).current;

    useEffect(() => {
        if (phase !== 'calling') return;

        Animated.parallel([
            Animated.spring(callingScale, {
                toValue: 1,
                friction: 5,
                tension: 80,
                useNativeDriver: true,
            }),
            Animated.timing(callingOpacity, {
                toValue: 1,
                duration: 400,
                useNativeDriver: true,
            }),
        ]).start();
    }, [phase]);

    const handleCancel = () => {
        router.back();
    };

    return (
        <View style={styles.container}>
            {/* SOS alert card */}
            <View style={styles.cardOuter}>
                <View style={styles.card}>
                    <Text style={styles.alertTitle}>
                        ALERT <Text style={styles.alertSOS}>SOS</Text> IN
                    </Text>

                    <Text style={styles.alertSub}>
                        your location and live footage will be{'\n'}shared with 5 trusted contacts in
                    </Text>

                    {/* Countdown / Calling */}
                    <View style={styles.countdownArea}>
                        {phase === 'countdown' ? (
                            <Animated.Text
                                style={[
                                    styles.countdownText,
                                    {
                                        transform: [{ scale: scaleAnim }],
                                        opacity: opacityAnim,
                                    },
                                ]}
                            >
                                {count}
                            </Animated.Text>
                        ) : (
                            <PulseAnimation duration={1500} minScale={0.95} maxScale={1.05}>
                                <Animated.Text
                                    style={[
                                        styles.callingText,
                                        {
                                            transform: [{ scale: callingScale }],
                                            opacity: callingOpacity,
                                        },
                                    ]}
                                >
                                    calling help...
                                </Animated.Text>
                            </PulseAnimation>
                        )}
                    </View>
                </View>
            </View>

            {/* Cancel button */}
            <TouchableOpacity
                style={styles.cancelButton}
                onPress={handleCancel}
                activeOpacity={0.85}
            >
                <PulseAnimation
                    active={phase === 'countdown'}
                    duration={800}
                    minScale={0.95}
                    maxScale={1.05}
                >
                    <View style={styles.cancelInner}>
                        <Text style={styles.cancelText}>cancel SOS</Text>
                    </View>
                </PulseAnimation>
            </TouchableOpacity>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: 'rgba(0, 0, 0, 0.85)',
        justifyContent: 'center',
        alignItems: 'center',
        paddingHorizontal: Spacing.xl,
    },
    cardOuter: {
        width: '100%',
        borderRadius: BorderRadius.lg + 4,
        borderWidth: 3,
        borderColor: Colors.sosRed,
        overflow: 'hidden',
        shadowColor: Colors.sosRed,
        shadowOffset: { width: 0, height: 0 },
        shadowOpacity: 0.4,
        shadowRadius: 20,
        elevation: 15,
    },
    card: {
        backgroundColor: Colors.white,
        borderRadius: BorderRadius.lg,
        paddingVertical: Spacing.xl,
        paddingHorizontal: Spacing.lg,
        alignItems: 'center',
    },
    alertTitle: {
        fontSize: FontSizes.xl + 4,
        fontWeight: '900',
        color: Colors.bgDark,
        textAlign: 'center',
        marginBottom: Spacing.md,
    },
    alertSOS: {
        color: Colors.sosRed,
    },
    alertSub: {
        fontSize: FontSizes.sm,
        color: Colors.grayDark,
        textAlign: 'center',
        lineHeight: FontSizes.sm * 1.6,
        marginBottom: Spacing.lg,
    },
    countdownArea: {
        height: 160,
        justifyContent: 'center',
        alignItems: 'center',
        width: '100%',
    },
    countdownText: {
        fontSize: 120,
        fontWeight: '900',
        color: Colors.bgDark,
        textAlign: 'center',
    },
    callingText: {
        fontSize: FontSizes.xxl,
        fontWeight: '800',
        color: Colors.sosRed,
        textAlign: 'center',
    },
    cancelButton: {
        marginTop: Spacing.xl,
    },
    cancelInner: {
        backgroundColor: Colors.sosRed,
        borderRadius: BorderRadius.full,
        paddingVertical: Spacing.md,
        paddingHorizontal: Spacing.xxl,
    },
    cancelText: {
        fontSize: FontSizes.md,
        fontWeight: '700',
        color: Colors.white,
    },
});
