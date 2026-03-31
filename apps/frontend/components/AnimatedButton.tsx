// AnimatedButton.tsx
import { useRef, useCallback } from 'react';
import { Animated, Pressable, Easing, ViewStyle, StyleProp } from 'react-native';

type AnimatedButtonProps = {
    children: React.ReactNode;
    onPress?: () => void;
    depth?: number;
    style?: StyleProp<ViewStyle>;
};

export function AnimatedButton({
    children,
    onPress,
    depth = 4,
    style,
}: AnimatedButtonProps) {
    const anim = useRef(new Animated.Value(0)).current;

    const pressIn = useCallback(() => {
        Animated.timing(anim, {
            toValue: 1,
            duration: 70,
            easing: Easing.out(Easing.cubic),
            useNativeDriver: true,
        }).start();
    }, []);

    const pressOut = useCallback(() => {
        Animated.timing(anim, {
            toValue: 0,
            duration: 90,
            easing: Easing.out(Easing.quad),
            useNativeDriver: true,
        }).start();
    }, []);

    const translateY = anim.interpolate({
        inputRange: [0, 1],
        outputRange: [0, depth],
    });

    return (
        <Pressable onPressIn={pressIn} onPressOut={pressOut} onPress={onPress}>
            <Animated.View style={[style, { transform: [{ translateY }] }]}>
                {children}
            </Animated.View>
        </Pressable>
    );
}