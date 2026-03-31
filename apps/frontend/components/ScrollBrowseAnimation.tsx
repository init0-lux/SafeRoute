import React, { useRef } from 'react';
import { Animated, ScrollView, ViewStyle, StyleProp, Dimensions } from 'react-native';

const { width: SCREEN_WIDTH } = Dimensions.get('window');

interface ScrollBrowseAnimationProps {
    children: React.ReactNode[];
    style?: StyleProp<ViewStyle>;
    itemWidth?: number;
    spacing?: number;
}

export default function ScrollBrowseAnimation({
    children,
    style,
    itemWidth = SCREEN_WIDTH * 0.75,
    spacing = 16,
}: ScrollBrowseAnimationProps) {
    const scrollX = useRef(new Animated.Value(0)).current;

    return (
        <Animated.ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            snapToInterval={itemWidth + spacing}
            decelerationRate="fast"
            contentContainerStyle={[{ paddingHorizontal: spacing }]}
            onScroll={Animated.event(
                [{ nativeEvent: { contentOffset: { x: scrollX } } }],
                { useNativeDriver: true },
            )}
            scrollEventThrottle={16}
            style={style}
        >
            {children.map((child, index) => {
                const inputRange = [
                    (index - 1) * (itemWidth + spacing),
                    index * (itemWidth + spacing),
                    (index + 1) * (itemWidth + spacing),
                ];

                const scale = scrollX.interpolate({
                    inputRange,
                    outputRange: [0.9, 1, 0.9],
                    extrapolate: 'clamp',
                });

                const opacity = scrollX.interpolate({
                    inputRange,
                    outputRange: [0.6, 1, 0.6],
                    extrapolate: 'clamp',
                });

                const translateY = scrollX.interpolate({
                    inputRange,
                    outputRange: [10, 0, 10],
                    extrapolate: 'clamp',
                });

                return (
                    <Animated.View
                        key={index}
                        style={{
                            width: itemWidth,
                            marginRight: spacing,
                            transform: [{ scale }, { translateY }],
                            opacity,
                        }}
                    >
                        {child}
                    </Animated.View>
                );
            })}
        </Animated.ScrollView>
    );
}
